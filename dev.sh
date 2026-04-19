#!/bin/bash
#
# escrow-dev: Launch all backend services in a tmux session with colour-coded windows.
#
# Usage:
#   ./dev.sh          — start (or re-attach if already running)
#   ./dev.sh --kill   — stop the session and all services
#

SESSION="escrow-dev"

start() {
  # If session already exists, just attach
  if tmux has-session -t "$SESSION" 2>/dev/null; then
    tmux attach-session -t "$SESSION"
    return
  fi

  # Colour palette
  CLR_GATEWAY="#[fg=colour39]"       # blue
  CLR_SIMPLEBANK="#[fg=colour82]"    # green
  CLR_ESCROW="#[fg=colour226]"       # yellow
  CLR_PROFILE="#[fg=colour205]"      # magenta
  CLR_PAYMENT="#[fg=colour51]"       # cyan
  CLR_RESET="#[fg=colour250]"

  # Start window numbering from 1 instead of 0
  tmux set-option -g base-index 1

  # ── Create session with first window (gateway) ──────────────────────────────
  tmux new-session -d -s "$SESSION" -n "gateway" "
    echo 'Starting gateway...'
    cd gateway && go run main.go
  "

  tmux set-window-option -t "$SESSION:gateway"   window-status-format " #1: gateway "
  tmux set-window-option -t "$SESSION:gateway"   window-status-current-format " #1: $CLR_GATEWAY gateway $CLR_RESET "

  # ── Remaining services as additional windows ─────────────────────────────────
  tmux new-window -t "$SESSION" -n "simplebank"    "cd simplebank && go run main.go"
  tmux new-window -t "$SESSION" -n "escrow-bounty" "cd escrow-bounty && go run main.go"
  tmux new-window -t "$SESSION" -n "user-profile"  "cd user-profile-service && go run main.go"
  tmux new-window -t "$SESSION" -n "payment"        "cd payment-service && go run main.go"

  tmux set-window-option -t "$SESSION:simplebank"    window-status-current-format " #1: $CLR_SIMPLEBANK simplebank $CLR_RESET "
  tmux set-window-option -t "$SESSION:escrow-bounty" window-status-current-format " #1: $CLR_ESCROW escrow-bounty $CLR_RESET "
  tmux set-window-option -t "$SESSION:user-profile"  window-status-current-format " #1: $CLR_PROFILE user-profile $CLR_RESET "
  tmux set-window-option -t "$SESSION:payment"       window-status-current-format " #1: $CLR_PAYMENT payment $CLR_RESET "

  tmux set-window-option -g window-status-format " #1: #W "
  tmux set-window-option -g window-status-current-format " #1: #W "

  tmux attach-session -t "$SESSION"
}

stop() {
  tmux kill-session -t "$SESSION" 2>/dev/null
  echo "Session '$SESSION' stopped."
}

case "${1:-}" in
  --kill) stop ;;
  *)      start ;;
esac
