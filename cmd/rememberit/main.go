package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/kunalsin9h/rememberit/internal/adapters/notifier"
	"github.com/kunalsin9h/rememberit/internal/adapters/ollama"
	sqliteadapter "github.com/kunalsin9h/rememberit/internal/adapters/sqlite"
	"github.com/kunalsin9h/rememberit/internal/adapters/timeparser"
	"github.com/kunalsin9h/rememberit/internal/app"
	"github.com/kunalsin9h/rememberit/internal/domain"
	"github.com/kunalsin9h/rememberit/internal/ports"
	"github.com/spf13/cobra"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	dataDir, err := dataDirectory()
	if err != nil {
		return err
	}

	db, err := sqliteadapter.Open(filepath.Join(dataDir, "memories.db"))
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	cfg := &configHelper{cfg: db.Config}

	aiClient := ollama.New(
		cfg.get("ollama.url", "http://localhost:11434"),
		cfg.get("ollama.embed_model", "nomic-embed-text"),
		cfg.get("ollama.chat_model", "llama3.2:3b"),
	)

	var notif ports.NotifierPort
	if notifier.IsAvailable() {
		notif = notifier.NewNotifySend()
	} else {
		notif = notifier.NewStdout()
	}

	memorySvc := app.NewMemoryService(db.Store, aiClient, timeparser.New())
	reminderSvc := app.NewReminderService(db.Store, notif)

	root := &cobra.Command{
		Use:   "rememberit",
		Short: "AI-native terminal memory and reminder system",
		Long: `rememberit — save anything from your terminal, recall it with natural language.

Examples:
  rememberit add "claude --resume abc123" --for "rememberit build session"
  rememberit add "book conference ticket" --remind "in 30 minutes"
  rememberit ask "which claude session was I building rememberit in?"
  rememberit list`,
	}

	root.AddCommand(
		addCmd(memorySvc),
		askCmd(memorySvc),
		listCmd(memorySvc),
		getCmd(memorySvc),
		deleteCmd(memorySvc),
		checkCmd(reminderSvc),
		daemonCmd(reminderSvc, cfg),
		configCmd(db.Config),
	)

	return root.Execute()
}

// --- add ---

func addCmd(svc *app.MemoryService) *cobra.Command {
	var forLabel, remind, typeHint string
	var tags []string

	cmd := &cobra.Command{
		Use:   "add <content>",
		Short: "Save a new memory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := svc.Add(context.Background(), app.AddRequest{
				Content:    args[0],
				ForLabel:   forLabel,
				RemindExpr: remind,
				TypeHint:   domain.MemoryType(typeHint),
				ExtraTags:  tags,
			})
			if err != nil {
				return err
			}

			fmt.Printf("saved    %s\n", shortID(m.ID))
			fmt.Printf("type     %s\n", m.Type)
			if len(m.Tags) > 0 {
				fmt.Printf("tags     %s\n", strings.Join(m.Tags, ", "))
			}
			if m.RemindAt != nil {
				fmt.Printf("remind   %s\n", relTime(*m.RemindAt))
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&forLabel, "for", "f", "", "Context: why are you saving this?")
	cmd.Flags().StringVar(&remind, "remind", "", `When to remind you, e.g. "in 30 minutes", "tomorrow 9am"`)
	cmd.Flags().StringVar(&typeHint, "type", "", "Override type detection: command|note|reminder|url|fact")
	cmd.Flags().StringArrayVar(&tags, "tag", nil, "Add a tag (repeatable: --tag docker --tag networking)")
	return cmd
}

// --- ask ---

func askCmd(svc *app.MemoryService) *cobra.Command {
	return &cobra.Command{
		Use:   "ask <question>",
		Short: "Query your memories with natural language",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			answer, err := svc.Ask(context.Background(), strings.Join(args, " "))
			if err != nil {
				return err
			}
			fmt.Println(answer)
			return nil
		},
	}
}

// --- list ---

func listCmd(svc *app.MemoryService) *cobra.Command {
	var typeFlag, tagFlag string
	var limit int
	var remindOnly bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List saved memories",
		RunE: func(cmd *cobra.Command, args []string) error {
			memories, err := svc.List(context.Background(), domain.ListFilter{
				Type:          domain.MemoryType(typeFlag),
				Tag:           tagFlag,
				Limit:         limit,
				OnlyReminders: remindOnly,
			})
			if err != nil {
				return err
			}
			if len(memories) == 0 {
				fmt.Println("no memories found")
				return nil
			}
			printTable(memories)
			return nil
		},
	}

	cmd.Flags().StringVar(&typeFlag, "type", "", "Filter by type: command|note|reminder|url|fact")
	cmd.Flags().StringVar(&tagFlag, "tag", "", "Filter by tag")
	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum results")
	cmd.Flags().BoolVar(&remindOnly, "remind", false, "Show only pending reminders")
	return cmd
}

// --- get ---

func getCmd(svc *app.MemoryService) *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Show full details of a memory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := svc.GetByID(context.Background(), args[0])
			if err != nil {
				return err
			}
			printDetail(m)
			return nil
		},
	}
}

// --- delete ---

func deleteCmd(svc *app.MemoryService) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a memory by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !force {
				fmt.Printf("delete %s? [y/N] ", shortID(args[0]))
				var ans string
				fmt.Scanln(&ans) //nolint:errcheck
				if strings.ToLower(strings.TrimSpace(ans)) != "y" {
					fmt.Println("cancelled")
					return nil
				}
			}
			if err := svc.Delete(context.Background(), args[0]); err != nil {
				return err
			}
			fmt.Printf("deleted %s\n", shortID(args[0]))
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "y", false, "Skip confirmation prompt")
	return cmd
}

// --- check (designed for PROMPT_COMMAND) ---

func checkCmd(svc *app.ReminderService) *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "Check for due reminders (silent unless reminders are firing)",
		Long: `Designed to run on every shell prompt via PROMPT_COMMAND:

  Add to ~/.bashrc or ~/.zshrc:
    export PROMPT_COMMAND="rememberit check; $PROMPT_COMMAND"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return svc.CheckAndFire(context.Background())
		},
	}
}

// --- daemon ---

func daemonCmd(svc *app.ReminderService, cfg *configHelper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "daemon",
		Short: "Manage the background reminder daemon",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "start",
			Short: "Start the daemon in the foreground (use systemd for background)",
			RunE: func(cmd *cobra.Command, args []string) error {
				intervalStr := cfg.get("reminder.poll_interval", "30s")
				interval, err := time.ParseDuration(intervalStr)
				if err != nil {
					interval = 30 * time.Second
				}
				ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
				defer stop()
				fmt.Printf("daemon started  poll-interval=%s  Ctrl+C to stop\n", interval)
				return svc.RunDaemon(ctx, interval)
			},
		},
		&cobra.Command{
			Use:   "install",
			Short: "Install as a systemd user service",
			RunE: func(cmd *cobra.Command, args []string) error {
				return installSystemdService()
			},
		},
	)
	return cmd
}

// --- config ---

func configCmd(cfg ports.ConfigPort) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage rememberit configuration",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "set <key> <value>",
			Short: "Set a config value",
			Args:  cobra.ExactArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				if err := cfg.Set(args[0], args[1]); err != nil {
					return err
				}
				fmt.Printf("set %s = %s\n", args[0], args[1])
				return nil
			},
		},
		&cobra.Command{
			Use:   "get <key>",
			Short: "Get a config value",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				v, err := cfg.Get(args[0])
				if err != nil {
					return err
				}
				if v == "" {
					fmt.Println("(not set)")
				} else {
					fmt.Println(v)
				}
				return nil
			},
		},
		&cobra.Command{
			Use:   "list",
			Short: "List all config values",
			RunE: func(cmd *cobra.Command, args []string) error {
				all, err := cfg.All()
				if err != nil {
					return err
				}
				if len(all) == 0 {
					fmt.Println("(no config set — all defaults in use)")
					return nil
				}
				w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
				for k, v := range all {
					fmt.Fprintf(w, "%s\t%s\n", k, v)
				}
				return w.Flush()
			},
		},
	)
	return cmd
}

// --- output helpers ---

func printTable(memories []*domain.Memory) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTYPE\tCONTENT\tFOR\tCREATED")
	fmt.Fprintln(w, "--\t----\t-------\t---\t-------")
	for _, m := range memories {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			shortID(m.ID),
			m.Type,
			truncate(m.Content, 50),
			truncate(m.ForLabel, 30),
			relTime(m.CreatedAt),
		)
	}
	w.Flush()
}

func printDetail(m *domain.Memory) {
	fmt.Printf("ID       : %s\n", m.ID)
	fmt.Printf("Content  : %s\n", m.Content)
	if m.ForLabel != "" {
		fmt.Printf("For      : %s\n", m.ForLabel)
	}
	fmt.Printf("Type     : %s\n", m.Type)
	if len(m.Tags) > 0 {
		fmt.Printf("Tags     : %s\n", strings.Join(m.Tags, ", "))
	}
	fmt.Printf("Created  : %s (%s)\n", m.CreatedAt.Format(time.RFC822), relTime(m.CreatedAt))
	if m.WorkingDir != "" {
		fmt.Printf("Dir      : %s\n", m.WorkingDir)
	}
	if m.Hostname != "" {
		fmt.Printf("Host     : %s\n", m.Hostname)
	}
	if m.RemindAt != nil {
		fmt.Printf("Remind at: %s (%s)\n", m.RemindAt.Format(time.RFC822), relTime(*m.RemindAt))
	}
	if m.RemindedAt != nil {
		fmt.Printf("Reminded : %s\n", m.RemindedAt.Format(time.RFC822))
	}
}

func shortID(id string) string {
	if len(id) > 10 {
		return id[:10]
	}
	return id
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}

func relTime(t time.Time) string {
	d := time.Since(t)
	future := d < 0
	if future {
		d = -d
	}

	var s string
	switch {
	case d < time.Minute:
		s = "just now"
		if future {
			return "in a moment"
		}
		return s
	case d < time.Hour:
		s = fmt.Sprintf("%d min", int(d.Minutes()))
	case d < 24*time.Hour:
		s = fmt.Sprintf("%d hr", int(d.Hours()))
	default:
		s = fmt.Sprintf("%d days", int(d.Hours()/24))
	}

	if future {
		return "in " + s
	}
	return s + " ago"
}

// --- system helpers ---

func dataDirectory() (string, error) {
	var base string
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		base = xdg
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".local", "share")
	}
	dir := filepath.Join(base, "rememberit")
	return dir, os.MkdirAll(dir, 0o700)
}

func installSystemdService() error {
	execPath, err := os.Executable()
	if err != nil {
		return err
	}
	service := fmt.Sprintf(`[Unit]
Description=rememberit reminder daemon
After=graphical-session.target

[Service]
ExecStart=%s daemon start
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=default.target
`, execPath)

	svcDir := filepath.Join(os.Getenv("HOME"), ".config", "systemd", "user")
	if err := os.MkdirAll(svcDir, 0o755); err != nil {
		return err
	}
	svcPath := filepath.Join(svcDir, "rememberit.service")
	if err := os.WriteFile(svcPath, []byte(service), 0o644); err != nil {
		return err
	}
	fmt.Printf("installed: %s\n", svcPath)
	fmt.Println("enable  : systemctl --user enable --now rememberit")
	return nil
}

// --- configHelper ---
// Thin wrapper to read config values with inline defaults,
// used only during wiring in main — not part of the domain.

type configHelper struct {
	cfg ports.ConfigPort
}

func (c *configHelper) get(key, defaultVal string) string {
	v, err := c.cfg.Get(key)
	if err != nil || v == "" {
		return defaultVal
	}
	return v
}
