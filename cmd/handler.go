package cmd

import (
	"html"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/phuangpheth/feedback/feedback"
)

type handler struct {
	router *echo.Echo

	service *feedback.Service
}

// NewHandler creates a new handler.
func NewHandler(e *echo.Echo, service *feedback.Service) {
	h := &handler{
		router:  e,
		service: service,
	}
	app := h.router.Group("/api/v1")

	app.GET("/getting", h.Getting)

	app.GET("/questions", h.GetAllQuestion)
	app.POST("/questions", h.StoreQuestion)
	app.PUT("/questions/:id", h.UpdateQuestion)

	app.GET("/assessments", h.GetAllAssessment)
	app.POST("/feedbacks", h.StoreFeedback)
}

func (h *handler) Getting(c echo.Context) error {
	return c.JSON(http.StatusOK, echo.Map{
		"message": "Hello from echo v4",
	})
}

func (h *handler) StoreQuestion(c echo.Context) error {
	var q feedback.Question
	if err := c.Bind(&q); err != nil {
		return err
	}
	question, err := h.service.StoreQuestion(c.Request().Context(), &q)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, question)
}

func (h *handler) UpdateQuestion(c echo.Context) error {
	var q feedback.Question
	if err := c.Bind(&q); err != nil {
		return err
	}
	q.ID = c.Param("id")
	question, err := h.service.UpdateQuestion(c.Request().Context(), &q)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, question)
}

func (h *handler) GetAllQuestion(c echo.Context) error {
	questions, err := h.service.GetAllQuestion(c.Request().Context())
	if err != nil {
		return err
	}
	q := html.EscapeString(strings.TrimSpace(c.QueryParam("q")))
	if strings.ToLower(q) == "enable" {
		return c.JSON(http.StatusOK, questions.Enable())
	}
	return c.JSON(http.StatusOK, questions)
}

func (h *handler) GetAllAssessment(c echo.Context) error {
	assessments, err := h.service.GetAllAssessment(c.Request().Context())
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, assessments)
}

func (h *handler) StoreFeedback(c echo.Context) error {
	var req struct {
		Feedbacks []feedback.Feedback `json:"assessments"`
	}
	if err := c.Bind(&req); err != nil {
		return err
	}

	if err := h.service.BulkStoreFeedBack(c.Request().Context(), req.Feedbacks); err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, nil)
}
