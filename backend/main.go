// Command ss-tax-engine is the backend entrypoint. Subcommands:
//
//	(none) | serve   run the HTTP API
//	migrate          apply SQL migrations in ./migrations
//	seed             load the built-in tax rulesets into the tax_rules table
package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"ss-tax-engine/internal/api"
	"ss-tax-engine/internal/auth"
	"ss-tax-engine/internal/billing"
	"ss-tax-engine/internal/config"
	"ss-tax-engine/internal/db"
	"ss-tax-engine/internal/email"
	"ss-tax-engine/internal/engine"
)

func main() {
	cmd := "serve"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	ctx := context.Background()
	database, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer database.Close()

	switch cmd {
	case "migrate":
		if err := database.Migrate(ctx, migrationsDir()); err != nil {
			log.Fatalf("migrate: %v", err)
		}
		log.Println("migrations complete")
	case "seed":
		if err := seed(ctx, database); err != nil {
			log.Fatalf("seed: %v", err)
		}
		log.Println("seed complete")
	case "serve":
		serve(cfg, database)
	default:
		log.Fatalf("unknown command %q", cmd)
	}
}

func serve(cfg *config.Config, database *db.DB) {
	a := auth.New(cfg.JWTSecret)
	// Production could swap DefaultProvider() for a DB-backed provider that reads
	// tax_rules; the built-in provider is the seed source of truth.
	e := engine.New(engine.DefaultProvider())
	mailer := email.New(cfg.ResendAPIKey, cfg.EmailFromAddr, cfg.AppBaseURL)
	bill := billing.New(cfg.DodoAPIKey, cfg.DodoWebhookSecret, cfg.DodoProProductID, cfg.DodoEnv)
	srv := api.NewServer(database, a, e, mailer, bill, cfg.AppBaseURL)
	log.Printf("listening on :%s", cfg.Port)
	if err := srv.Router().Run(":" + cfg.Port); err != nil {
		log.Fatalf("server: %v", err)
	}
}

// seed writes the built-in rulesets into tax_rules, one row per jurisdiction.
func seed(ctx context.Context, database *db.DB) error {
	provider := engine.DefaultProvider()
	for year, rs := range provider.Years {
		fedJSON, _ := json.Marshal(rs.Federal)
		if err := database.UpsertTaxRule(ctx, year, "FEDERAL", fedJSON); err != nil {
			return err
		}
		for code, sr := range rs.States {
			b, _ := json.Marshal(sr)
			if err := database.UpsertTaxRule(ctx, year, code, b); err != nil {
				return err
			}
		}
		log.Printf("seeded %d: FEDERAL + %d states", year, len(rs.States))
	}
	return nil
}

func migrationsDir() string {
	if d := os.Getenv("MIGRATIONS_DIR"); d != "" {
		return d
	}
	return "migrations"
}
