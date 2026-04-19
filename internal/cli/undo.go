package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/flaticols/semtag/internal/git"
	"github.com/flaticols/semtag/internal/tui"
)

func runUndo(cfg *Config, repo git.Backend) error {
	ver, err := repo.LatestTag(cfg.Prefix)
	if err != nil {
		var tagErr git.SemVerTagError
		if errors.As(err, &tagErr) {
			if tagErr.NoTags {
				slog.Error("no tags found to remove")
				os.Exit(1)
			}
			slog.Error(fmt.Sprintf("tag '%s' is not a valid semver tag", tagErr.Tag))
			os.Exit(1)
		}
		return err
	}

	tag := formatTag(ver.Stringv(), cfg.Prefix)

	confirm := tui.Confirm("Are you sure?",
		tui.WithYes(fmt.Sprintf("Yes remove %s!", tag)),
		tui.BypassIf(cfg.Brave, true),
		tui.Interactive(cfg.Interactive),
	)

	if !confirm {
		return nil
	}

	slog.Info(fmt.Sprintf("removing tag %s", tag))

	if err := repo.RemoveTag(tag); err != nil {
		return err
	}
	slog.Info("local tag removed")

	if !cfg.Local {
		if err := repo.RemoveRemoteTag(tag); err != nil {
			slog.Error(fmt.Sprintf("remote tag not removed: %s", err))
			os.Exit(1)
		}
		slog.Info("remote tag removed")
	}

	if cfg.JSON {
		json.NewEncoder(os.Stdout).Encode(map[string]string{"removed": tag})
	}

	return nil
}
