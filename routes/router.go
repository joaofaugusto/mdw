package routes

import (
	"html/template"
	"net/http"
	"sync"

	"github.com/gin-contrib/cache"
	"github.com/gin-contrib/cache/persistence"
	"github.com/gin-gonic/gin"
)

func HealthCheckHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

var (
	templateCache    map[string]*template.Template
	templateCacheMux sync.RWMutex
	store            persistence.CacheStore
)

func init() {
	// Initialize template cache
	templateCache = make(map[string]*template.Template)
	// Create a memory store for caching rendered pages
	store = persistence.NewInMemoryStore(60) // Cache for 60 seconds
}

// preloadTemplates parses all templates during initialization
func preloadTemplates() error {
	templateCacheMux.Lock()
	defer templateCacheMux.Unlock()

	templates, err := template.ParseGlob("public/*.tmpl.html")
	if err != nil {
		return err
	}

	for _, t := range templates.Templates() {
		templateCache[t.Name()] = t
	}
	return nil
}

func renderizarTemplate(templateName string) gin.HandlerFunc {
	// Use template caching
	return cache.CachePage(store, 60, func(c *gin.Context) {
		templateCacheMux.RLock()
		tmpl, exists := templateCache[templateName]
		templateCacheMux.RUnlock()

		if !exists {
			c.String(http.StatusInternalServerError, "Template not found")
			return
		}

		c.Header("Content-Type", "text/html")
		c.Status(http.StatusOK)
		tmpl.Execute(c.Writer, nil)
	})
}

// Essa função permite que crie vários servers com diferentes portas (sempre checar o .env)
func criarRouter() (http.Handler, error) {
	// Parse templates during router creation
	if err := preloadTemplates(); err != nil {
		return nil, err
	}

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	// Use more performant middlewares
	router.Use(
		gin.Recovery(),
		gin.Logger(),
	)

	// Routes
	router.GET("/", renderizarTemplate("index.tmpl.html"))
	router.GET("/health_check", HealthCheckHandler)

	return router, nil
}

// Inicialização dos servers retornando a mensagem deles
func MdwRouter_01() (http.Handler, error) {
	return criarRouter()
}

func MdwRouter_02() (http.Handler, error) {
	return criarRouter()
}
