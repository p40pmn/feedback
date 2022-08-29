package feedback

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/phuangpheth/feedback/database"
	"github.com/shopspring/decimal"
)

var (
	// ErrQuestionUnknown is returned when question is not found.
	ErrQuestionUnknown = errors.New("unknown question")
)

// Service represents a feedback service.
type Service struct {
	db *database.DB
}

// NewService creates a new feedback service.
func NewService(db *database.DB) *Service {
	return &Service{
		db: db,
	}
}

func (s *Service) GetAllQuestion(ctx context.Context) (Questions, error) {
	questions, err := findAllQuestion(ctx, s.db)
	if err != nil {
		return nil, err
	}
	return questions, nil
}

func (s *Service) StoreQuestion(ctx context.Context, q *Question) (*Question, error) {
	q.ID = genID()
	q.UpdatedAt = time.Now()
	if err := storeQuestion(ctx, s.db, q); err != nil {
		return nil, err
	}
	return q, nil
}

func (s *Service) UpdateQuestion(ctx context.Context, q *Question) (*Question, error) {
	question, err := findQuestionByID(ctx, s.db, q.ID)
	if err != nil {
		return nil, err
	}
	question.Edit(q.Title, q.Enable)
	if err := updateQuestion(ctx, s.db, question); err != nil {
		return nil, err
	}
	return question, nil
}

func (s *Service) BulkStoreFeedBack(ctx context.Context, feedbacks []Feedback) error {
	if err := bulkStoreFeedback(ctx, s.db, feedbacks); err != nil {
		return err
	}
	return nil
}

func (s *Service) GetAllAssessment(ctx context.Context) ([]Assessment, error) {
	as, err := findAllAssessment(ctx, s.db)
	if err != nil {
		return nil, err
	}
	return as, nil
}

type Questions []Question

func (qs Questions) Enable() Questions {
	q := make(Questions, 0)
	for _, v := range qs {
		if v.Enable {
			q = append(q, v)
		}
	}
	return q
}

type Question struct {
	Enable    bool      `json:"enable"`
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	UpdatedBy string    `json:"updatedBy"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (q *Question) Edit(title string, enable bool) {
	q.Title = title
	q.Enable = enable
	q.UpdatedBy = ""
	q.UpdatedAt = time.Now()
}

type Feedback struct {
	ID         string          `json:"id"`
	TeachingID string          `json:"teachingId" validate:"required"`
	QuestionID string          `json:"questionId" validate:"required"`
	Rating     decimal.Decimal `json:"rating"`
}

type Assessment struct {
	TeachingID string          `json:"TeachingId"`
	Rating     decimal.Decimal `json:"rating"`
}

var questionColumns = []string{
	"id",
	"title",
	"is_display",
	"updated_by",
	"updated_at",
}

func storeQuestion(ctx context.Context, db *database.DB, q *Question) error {
	query, args, err := sq.Insert("questions").
		Columns(questionColumns...).
		Values(
			q.ID,
			q.Title,
			q.Enable,
			q.UpdatedBy,
			q.UpdatedAt,
		).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return err
	}
	if _, err := db.Exec(ctx, query, args...); err != nil {
		return err
	}
	return nil
}

func updateQuestion(ctx context.Context, db *database.DB, q *Question) error {
	query, args, err := sq.Update("questions").
		Set("title", q.Title).
		Set("is_display", q.Enable).
		Set("updated_by", q.UpdatedBy).
		Set("updated_at", q.UpdatedAt).
		Where(sq.Eq{"id": q.ID}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return err
	}
	if _, err := db.Exec(ctx, query, args...); err != nil {
		return err
	}
	return nil
}

func findAllQuestion(ctx context.Context, db *database.DB) (Questions, error) {
	query, args, err := sq.Select(questionColumns...).
		From("questions").
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return nil, err
	}

	qs := make(Questions, 0)
	collection := func(rows *sql.Rows) error {
		q, err := scanQuestion(rows.Scan)
		if err != nil {
			return err
		}
		qs = append(qs, q)
		return nil
	}
	return qs, db.RunQuery(ctx, query, collection, args...)
}

func findQuestionByID(ctx context.Context, db *database.DB, id string) (*Question, error) {
	query, args, err := sq.Select(questionColumns...).
		From("questions").
		Where(sq.Eq{"id": id}).
		Limit(1).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return nil, err
	}

	row := db.QueryRow(ctx, query, args...)
	q, err := scanQuestion(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrQuestionUnknown
	}
	if err != nil {
		return nil, err
	}
	return &q, nil
}

func scanQuestion(scan func(...any) error) (q Question, _ error) {
	return q, scan(
		&q.ID,
		&q.Title,
		&q.Enable,
		&q.UpdatedBy,
		&q.UpdatedAt,
	)
}

var feedbackColumns = []string{
	"id",
	"teaching_id",
	"question_id",
	"rating",
}

func bulkStoreFeedback(ctx context.Context, db *database.DB, fs []Feedback) error {
	values := make([]any, 0)
	for _, v := range fs {
		values = append(values,
			genID(),
			v.TeachingID,
			v.QuestionID,
			v.Rating,
		)
	}
	return db.BulkInsert(ctx, "feedback_remarks", feedbackColumns, values, "")
}

func findAllAssessment(ctx context.Context, db *database.DB) ([]Assessment, error) {
	query, args, err := sq.Select(
		"teaching_id",
		"SUM(rating)/COUNT(id)",
	).
		From("feedback_remarks").
		GroupBy("teaching_id").
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return nil, err
	}

	as := make([]Assessment, 0)
	collection := func(rows *sql.Rows) error {
		a, err := scanAssessment(rows.Scan)
		if err != nil {
			return err
		}
		as = append(as, a)
		return nil
	}
	return as, db.RunQuery(ctx, query, collection, args...)
}

func scanAssessment(scan func(...any) error) (a Assessment, _ error) {
	return a, scan(
		&a.TeachingID,
		&a.Rating,
	)
}

// genID is a helper function to generate a unique ID with 8 characters.
func genID() string {
	byt := make([]byte, 4)
	if _, err := rand.Read(byt); err != nil {
		panic(err)
	}
	return strings.ToUpper(hex.EncodeToString(byt))
}
