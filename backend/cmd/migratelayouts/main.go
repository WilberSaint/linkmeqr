// Command migratelayouts backfills the print-card element tree for every
// card created before the designer was rewritten around one.
//
// It is idempotent: a card that already has a layout is skipped, so it is
// safe to run repeatedly and safe to run again after a partial failure. It
// writes each backfilled tree as revision 1, exactly as if an admin had
// saved it, so restore-a-previous-version works on migrated cards too.
//
//	go run ./cmd/migratelayouts            # backfill
//	go run ./cmd/migratelayouts -dry-run   # report only, write nothing
//	go run ./cmd/migratelayouts -force     # re-seed even cards that have a tree
//
// -force exists for the narrow case of re-running the backfill after fixing
// a bug in the seeder, before anyone has hand-edited the migrated cards. It
// DISCARDS hand edits, so it prompts unless -yes is also given.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"linkmeqr/backend/internal/config"
	"linkmeqr/backend/internal/database"
	"linkmeqr/backend/internal/models"
	"linkmeqr/backend/internal/repository"
	"linkmeqr/backend/internal/services"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "report what would change without writing anything")
	force := flag.Bool("force", false, "re-seed cards that already have a layout (DISCARDS hand edits)")
	yes := flag.Bool("yes", false, "skip the confirmation prompt for -force")
	flag.Parse()

	_ = godotenv.Load()          // backend/.env, if running from backend/
	_ = godotenv.Load("../.env") // repo-root .env, if running from backend/
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := database.Connect(cfg.MySQLDSN())
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer db.Close()

	if *force && !*dryRun && !*yes {
		fmt.Print("-force will overwrite hand-edited card designs. Type 'yes' to continue: ")
		answer, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		if strings.TrimSpace(answer) != "yes" {
			log.Fatal("aborted")
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	cardRepo := repository.NewPrintCardRepository(db)
	profileRepo := repository.NewProfileRepository(db)
	themeRepo := repository.NewProfileThemeRepository(db)

	cards, err := allPrintCards(ctx, db)
	if err != nil {
		log.Fatalf("load cards: %v", err)
	}
	log.Printf("found %d print cards", len(cards))

	var seeded, skipped, failed int
	for i := range cards {
		card := &cards[i]

		if card.Layout != nil && *card.Layout != "" && *card.Layout != "null" && !*force {
			skipped++
			continue
		}

		profile, err := profileRepo.GetByUserID(ctx, card.UserID)
		if err != nil {
			// A card whose owner has no profile can't resolve a theme or a
			// logo. It still gets a tree — just one built on the renderer's
			// own defaults — rather than being left un-migrated.
			log.Printf("card %s: no profile (%v), seeding with defaults", card.ID, err)
			profile = nil
		}

		var theme *models.ProfileTheme
		hasLogo, logoShape := false, ""
		if profile != nil {
			theme, _ = themeRepo.GetByProfileID(ctx, profile.ID)
			hasLogo = profile.LogoMediaID != nil
			if theme != nil {
				logoShape = theme.LogoShape
			}
		}

		layout := services.SeedCardLayout(card, theme, hasLogo, logoShape)
		if err := layout.Validate(); err != nil {
			log.Printf("card %s: seeded layout is invalid: %v", card.ID, err)
			failed++
			continue
		}
		encoded, err := json.Marshal(layout)
		if err != nil {
			log.Printf("card %s: encode: %v", card.ID, err)
			failed++
			continue
		}

		if *dryRun {
			log.Printf("card %s (%s/%s): would seed %d elements, %d bytes",
				card.ID, card.LayoutKey, card.SizePreset, len(layout.Elements), len(encoded))
			seeded++
			continue
		}

		if _, err := cardRepo.SaveLayout(ctx, card.ID, string(encoded), nil); err != nil {
			log.Printf("card %s: save: %v", card.ID, err)
			failed++
			continue
		}
		seeded++
	}

	verb := "seeded"
	if *dryRun {
		verb = "would seed"
	}
	log.Printf("done: %s %d, skipped %d (already had a layout), failed %d", verb, seeded, skipped, failed)
	if failed > 0 {
		os.Exit(1)
	}
}

// allPrintCards reads every card in the system. Deliberately a plain
// full-table scan rather than a repository method: this is a one-shot
// backfill over a table with at most a few thousand rows, and there is no
// reason for the application to grow an "every card, across all clients"
// query it would never otherwise use.
func allPrintCards(ctx context.Context, db interface {
	SelectContext(ctx context.Context, dest any, query string, args ...any) error
}) ([]models.PrintCard, error) {
	cards := []models.PrintCard{}
	err := db.SelectContext(ctx, &cards, `SELECT * FROM print_cards ORDER BY created_at`)
	return cards, err
}
