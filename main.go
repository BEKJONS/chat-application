package main

import (
	"chat_app/config"
	"chat_app/db"
	"chat_app/routes"
	"chat_app/services"
	"context"
	"log"
	"net/http"

	"go.uber.org/fx"
)

func main() {
	app := fx.New(
		fx.Provide(
			config.NewConfig,
			db.NewDB,
			services.NewAuthService,
			services.NewWebSocketService,
			services.NewSchedulerService,
			services.NewGroupService,
			routes.NewRoutes,
		),
		fx.Invoke(func(r *routes.Routes, ss *services.SchedulerService, lc fx.Lifecycle) {
			router := r.SetupRoutes()
			log.Println("Server starting on :8080")

			lc.Append(fx.Hook{
				OnStop: func(ctx context.Context) error {
					ss.Stop()
					log.Println("Scheduler stopped")
					return nil
				},
			})

			log.Fatal(http.ListenAndServe(":8080", router))
		}),
	)
	app.Run()
}
