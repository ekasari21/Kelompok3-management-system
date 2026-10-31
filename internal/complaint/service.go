package complaint

import (
	"errors"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}

type CreateComplaintRequest struct {
	UserID      uint    `json:"userID" form:"userID" binding:"required"`
	AssetID     uint    `json:"assetID" form:"assetID" binding:"required"`
	Title       string  `json:"title" form:"title" binding:"required"`
	Description string  `json:"description" form:"description" binding:"required"`
	Priority    float64 `json:"priority" form:"priority"`
	Status      string  `json:"status" form:"status"`
	Note        string  `json:"note" form:"note"`
	ImagePath   string  `json:"-" form:"-"`
}

type UpdateComplaintRequest struct {
	UserID      uint    `json:"userID" form:"userID" binding:"required"`
	AssetID     uint    `json:"assetID" form:"assetID" binding:"required"`
	Title       string  `json:"title" form:"title" binding:"required"`
	Description string  `json:"description" form:"description" binding:"required"`
	Priority    float64 `json:"priority" form:"priority"`
	Status      string  `json:"status" form:"status" binding:"required"`
	Note        string  `json:"note" form:"note"`
	IsFixed     bool    `json:"is_fixed" form:"is_fixed"`
	ImagePath   string  `json:"-" form:"-"`
}

func (s *Service) Create(req CreateComplaintRequest) (*Complaint, error) {
	if req.Priority < 0 || req.Priority > 1 {
		return nil, errors.New("priority harus antara 0 dan 1")
	}

	status := req.Status
	if status == "" {
		status = "low"
	}

	if status != "low" && status != "medium" &&
		status != "high" && status != "critical" {
		return nil, errors.New("status harus low, medium, high, atau critical")
	}

	now := time.Now()
	year := now.Year()

	code, err := s.repository.GenerateUniqueCode(year)
	if err != nil {
		log.Printf("[COMPLAINT] GenerateUniqueCode gagal: %v", err)
		return nil, err
	}
	log.Printf("[COMPLAINT] Kode complaint yang dibuat: %s", code)

	c := &Complaint{
		ID:          uuid.New(),
		Code:        code,
		UserID:      req.UserID,
		AssetID:     req.AssetID,
		Title:       req.Title,
		Description: req.Description,
		Priority:    req.Priority,
		Status:      status,
		Note:        req.Note,
		IsFixed:     false,
		ImagePath:   req.ImagePath,
	}

	if err := s.repository.Create(c); err != nil {
		return nil, err
	}

	return c, nil
}

func (s *Service) FindAll() ([]Complaint, error) {
	return s.repository.FindAll()
}

func (s *Service) FindByUserID(userID uint) ([]Complaint, error) {
	if userID == 0 {
		return nil, errors.New("User ID tidak valid")
	}

	return s.repository.FindByUserID(userID)
}

func (s *Service) FindByID(id string) (*Complaint, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.New("format UUID tidak valid")
	}

	return s.repository.FindByID(uid)
}

func (s *Service) Update(id string, req UpdateComplaintRequest) (*Complaint, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.New("format UUID tidak valid")
	}

	if req.Priority < 0 || req.Priority > 1 {
		return nil, errors.New("priority harus antara 0 dan 1")
	}

	req.Status = strings.ToLower(strings.TrimSpace(req.Status))
	if req.IsFixed {
		req.Status = "resolved"
	} else if req.Status != "low" && req.Status != "medium" &&
		req.Status != "high" && req.Status != "critical" {
		return nil, errors.New("status harus low, medium, high, atau critical")
	}

	c, err := s.repository.FindByID(uid)
	if err != nil {
		return nil, err
	}

	oldImagePath := c.ImagePath

	c.UserID = req.UserID
	c.AssetID = req.AssetID
	c.Title = req.Title
	c.Description = req.Description
	c.Priority = req.Priority
	c.Status = req.Status
	c.Note = req.Note
	c.IsFixed = req.IsFixed

	if req.ImagePath != "" {
		c.ImagePath = req.ImagePath
	}

	if c.IsFixed {
		c.Status = "resolved"
	}

	if err := s.repository.Update(c); err != nil {
		return nil, err
	}

	// Hapus file gambar lama setelah gambar baru berhasil disimpan.
	if req.ImagePath != "" && oldImagePath != "" && oldImagePath != req.ImagePath {
		removeComplaintImage(oldImagePath)
	}

	return c, nil
}

func (s *Service) Delete(id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return errors.New("format UUID tidak valid")
	}

	c, err := s.repository.FindByID(uid)
	if err != nil {
		return err
	}

	if err := s.repository.Delete(c.ID); err != nil {
		return err
	}

	removeComplaintImage(c.ImagePath)
	return nil
}

// Mempermudah handler membedakan data tidak ditemukan.
func IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

func (s *Service) MarkAsUnrepairable(id string, note string) (*Complaint, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.New("format UUID tidak valid")
	}

	return s.repository.MarkAsUnrepairable(uid, note)
}
