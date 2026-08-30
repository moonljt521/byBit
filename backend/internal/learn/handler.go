package learn

import (
	"net/http"

	"cryptosim/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Coins(c *gin.Context) {
	items, err := h.svc.Coins()
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "内容加载失败")
		return
	}
	response.OK(c, items)
}

func (h *Handler) Coin(c *gin.Context) {
	doc, err := h.svc.Coin(c.Param("slug"))
	if err != nil {
		response.Fail(c, http.StatusNotFound, response.CodeInvalidParams, "内容不存在")
		return
	}
	response.OK(c, doc)
}

func (h *Handler) Concepts(c *gin.Context) {
	items, err := h.svc.Concepts()
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "内容加载失败")
		return
	}
	response.OK(c, items)
}

func (h *Handler) Concept(c *gin.Context) {
	doc, err := h.svc.Concept(c.Param("slug"))
	if err != nil {
		response.Fail(c, http.StatusNotFound, response.CodeInvalidParams, "内容不存在")
		return
	}
	response.OK(c, doc)
}

func (h *Handler) Glossary(c *gin.Context) {
	terms, err := h.svc.Glossary()
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "内容加载失败")
		return
	}
	response.OK(c, terms)
}
