package repository

import (
	"encoding/json"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/fuzumoe/linkTorch-api/internal/model"
)

type URLRepository interface {
	Create(u *model.URL) error
	FindByID(id uint) (*model.URL, error)
	CountByUser(userID uint) (int, error)
	ListByUser(userID uint, p Pagination) ([]model.URL, error)
	Update(u *model.URL) error
	Delete(id uint) error
	UpdateStatus(id uint, status string) error
	SaveResults(id uint, res *model.AnalysisResult, links []model.Link) error
	Results(id uint) (*model.URL, error)
	ResultsWithDetails(id uint) (*model.URL, []*model.AnalysisResult, []*model.Link, error)
}

type urlRepo struct {
	db *gorm.DB
}

func NewURLRepo(db *gorm.DB) URLRepository {
	return &urlRepo{db: db}
}

func (r *urlRepo) CountByUser(userID uint) (int, error) {
	var count int64
	result := r.db.Model(&model.URL{}).Where("user_id = ?", userID).Count(&count)
	return int(count), result.Error
}
func (r *urlRepo) Create(u *model.URL) error {
	return r.db.Create(u).Error
}

func (r *urlRepo) FindByID(id uint) (*model.URL, error) {
	var u model.URL
	if err := r.db.
		Preload("AnalysisResults").
		Preload("Links").
		First(&u, id).
		Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *urlRepo) ListByUser(userID uint, p Pagination) ([]model.URL, error) {
	var urls []model.URL
	err := r.db.
		Where("user_id = ?", userID).
		Limit(p.Limit()).
		Offset(p.Offset()).
		Find(&urls).Error
	return urls, err
}

func (r *urlRepo) Update(u *model.URL) error {
	return r.db.Save(u).Error
}

func (r *urlRepo) Delete(id uint) error {
	res := r.db.Delete(&model.URL{}, id)
	if res.RowsAffected == 0 {
		return errors.New("url not found")
	}
	return res.Error
}

func (r *urlRepo) UpdateStatus(id uint, status string) error {
	return r.db.
		Model(&model.URL{}).
		Where("id = ?", id).
		Update("status", status).Error
}

func (r *urlRepo) SaveResults(id uint, res *model.AnalysisResult, links []model.Link) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		res.URLID = id
		if err := tx.Create(res).Error; err != nil {
			return err
		}
		for i := range links {
			links[i].URLID = id
		}
		return tx.CreateInBatches(&links, 500).Error
	})
}

func (r *urlRepo) Results(id uint) (*model.URL, error) {
	var u model.URL
	err := r.db.
		Preload("AnalysisResults").
		Preload("Links").
		First(&u, id).Error
	return &u, err
}

func (r *urlRepo) ResultsWithDetails(id uint) (*model.URL, []*model.AnalysisResult, []*model.Link, error) {

	var resultJSON string
	query := `SELECT
  JSON_OBJECT(
    'url',
      JSON_OBJECT(
        'id',           u.id,
        'user_id',      u.user_id,
        'original_url', u.original_url,
        'status',       u.status,
        'created_at',   DATE_FORMAT(u.created_at, '%Y-%m-%dT%H:%i:%s.%fZ'),
        'updated_at',   DATE_FORMAT(u.updated_at, '%Y-%m-%dT%H:%i:%s.%fZ')
      ),
    'analysis_results',
      (
        SELECT JSON_ARRAYAGG(
                 JSON_OBJECT(
                   'id',                  ar.id,
                   'url_id',              ar.url_id,
                   'html_version',        ar.html_version,
                   'title',               ar.title,
                   'h1_count',            ar.h1_count,
                   'h2_count',            ar.h2_count,
                   'h3_count',            ar.h3_count,
                   'h4_count',            ar.h4_count,
                   'h5_count',            ar.h5_count,
                   'h6_count',            ar.h6_count,
                   'has_login_form',      IF(ar.has_login_form = 1, CAST('true' AS JSON), CAST('false' AS JSON)),
                   'internal_link_count', ar.internal_link_count,
                   'external_link_count', ar.external_link_count,
                   'broken_link_count',   ar.broken_link_count,
                   'created_at',          DATE_FORMAT(ar.created_at, '%Y-%m-%dT%H:%i:%s.%fZ'),
                   'updated_at',          DATE_FORMAT(ar.updated_at, '%Y-%m-%dT%H:%i:%s.%fZ')
                 )
               )
        FROM   analysis_results ar
        WHERE  ar.url_id = u.id
        ORDER BY ar.created_at DESC
      ),
    'links',
      (
        SELECT JSON_ARRAYAGG(
                 JSON_OBJECT(
                   'id',          l.id,
                   'url_id',      l.url_id,
                   'href',        l.href,
                   'is_external', IF(l.is_external = 1, CAST('true' AS JSON), CAST('false' AS JSON)),
                   'status_code', l.status_code
                 )
               )
        FROM   links l
        WHERE  l.url_id = u.id
      )
  ) AS result_document
FROM urls u
WHERE u.id = ?`
	err := r.db.Raw(query, id).Scan(&resultJSON).Error
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to execute complex query: %w", err)
	}

	var result struct {
		URL             model.URL               `json:"url"`
		AnalysisResults []*model.AnalysisResult `json:"analysis_results"`
		Links           []*model.Link           `json:"links"`
	}

	if err := json.Unmarshal([]byte(resultJSON), &result); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to parse JSON result: %w", err)
	}

	return &result.URL, result.AnalysisResults, result.Links, nil
}
