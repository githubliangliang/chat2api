#!/usr/bin/env bash
#
# Sub2API 个人版一键发布：本地构建 → 打包 → 免密上传 VPS → 重启 → 健康检查（失败自动回滚）
#
#   1) 首次：cp deploy/deploy.env.example deploy/deploy.env  并填上你的 VPS 信息
#   2) 配免密：deploy/deploy-remote.sh --setup-key     # 只需输入一次服务器密码
#   3) 以后每次发布：deploy/deploy-remote.sh           # 全程不用输入密码
#
# 远端已经在跑的服务只会被「换二进制 + 重启」，config.yaml / systemd unit 不会被覆盖。
#
set -Eeuo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd -- "$SCRIPT_DIR/.." && pwd)"

if [ -t 1 ]; then
  C_R=$'\033[0;31m'; C_G=$'\033[0;32m'; C_Y=$'\033[1;33m'; C_C=$'\033[0;36m'; C_N=$'\033[0m'
else
  C_R=''; C_G=''; C_Y=''; C_C=''; C_N=''
fi
info() { printf '%s==>%s %s\n' "$C_C" "$C_N" "$*"; }
ok()   { printf '%s[OK]%s %s\n' "$C_G" "$C_N" "$*"; }
warn() { printf '%s[!]%s %s\n' "$C_Y" "$C_N" "$*" >&2; }
die()  {
  printf '%s[X] %s%s\n' "$C_R" "$1" "$C_N" >&2
  shift || true
  for line in "$@"; do printf '    %s\n' "$line" >&2; done
  exit 1
}
req() { [ -n "${2:-}" ] || die "选项 $1 需要一个参数"; }

usage() {
  cat <<'EOF'
用法: deploy/deploy-remote.sh [选项]

常用:
  (无参数)              构建 + 打包 + 上传 + 重启 + 健康检查
  --setup-key           一次性安装 SSH 公钥到服务器（此步会要求输入一次密码）
  --skip-frontend       跳过前端构建，复用已有 backend/internal/web/dist
  --skip-build          完全跳过构建，直接发布现有 ./sub2api
  --package-only        只本地构建打包，不连服务器（产物在 dist/）
  --no-restart          只替换二进制，不重启服务
  --dry-run             只打印将要执行的步骤
  -y, --yes             跳过确认

覆盖配置（默认读 deploy/deploy.env）:
  -c, --config FILE     指定配置文件
  -H, --host HOST       服务器地址
  -u, --user USER       SSH 用户（默认 root）
  -p, --port PORT       SSH 端口（默认 22）
  -i, --key FILE        SSH 私钥
  -d, --dir DIR         远端安装目录（默认 /opt/sub2api）
  -s, --service NAME    systemd 服务名（默认 sub2api）
      --arch ARCH       目标架构 amd64|arm64（默认 amd64）
      --force-config    覆盖远端 data/config.yaml（旧文件会备份）
      --force-unit      覆盖远端 systemd unit
  -h, --help            显示本帮助
EOF
}
# ---------- 先扫一遍 --config，再加载配置文件 ----------
CONFIG_FILE=""
if [ "$#" -gt 0 ]; then
  ARGV=("$@")
  for ((i = 0; i < ${#ARGV[@]}; i++)); do
    case "${ARGV[i]}" in
      -c|--config) CONFIG_FILE="${ARGV[i + 1]:-}" ;;
      --config=*)  CONFIG_FILE="${ARGV[i]#*=}" ;;
    esac
  done
fi
[ -n "$CONFIG_FILE" ] || CONFIG_FILE="$SCRIPT_DIR/deploy.env"
if [ -f "$CONFIG_FILE" ]; then
  set -a
  # shellcheck source=/dev/null
  . "$CONFIG_FILE"
  set +a
fi

# ---------- 默认值（配置文件 / 环境变量 > 这里）----------
: "${REMOTE_HOST:=}"
: "${REMOTE_USER:=root}"
: "${REMOTE_PORT:=22}"
: "${SSH_KEY:=}"
: "${SSH_PASSWORD:=}"
: "${SSH_STRICT:=accept-new}"
: "${REMOTE_DIR:=/opt/sub2api}"
: "${SERVICE_NAME:=sub2api}"
: "${SERVICE_USER:=sub2api}"
: "${APP_PORT:=8080}"
: "${RESTART_MODE:=systemd}"
: "${RESTART_COMMAND:=}"
: "${TARGET_ARCH:=amd64}"
: "${HEALTH_TIMEOUT:=60}"
: "${KEEP_RELEASES:=3}"
: "${ADMIN_EMAIL:=}"
: "${ADMIN_PASSWORD:=}"

SETUP_KEY=0; SKIP_FRONTEND=0; SKIP_BUILD=0; PACKAGE_ONLY=0
DRY_RUN=0; ASSUME_YES=0; FORCE_CONFIG=0; FORCE_UNIT=0; NO_RESTART=0

while [ "$#" -gt 0 ]; do
  case "$1" in
    -c|--config)     req "$@"; shift 2 ;;
    --config=*)      shift ;;
    -H|--host)       req "$@"; REMOTE_HOST="$2"; shift 2 ;;
    -u|--user)       req "$@"; REMOTE_USER="$2"; shift 2 ;;
    -p|--port)       req "$@"; REMOTE_PORT="$2"; shift 2 ;;
    -i|--key)        req "$@"; SSH_KEY="$2"; shift 2 ;;
    -d|--dir)        req "$@"; REMOTE_DIR="$2"; shift 2 ;;
    -s|--service)    req "$@"; SERVICE_NAME="$2"; shift 2 ;;
    --arch)          req "$@"; TARGET_ARCH="$2"; shift 2 ;;
    --setup-key)     SETUP_KEY=1; shift ;;
    --skip-frontend) SKIP_FRONTEND=1; shift ;;
    --skip-build)    SKIP_BUILD=1; SKIP_FRONTEND=1; shift ;;
    --package-only)  PACKAGE_ONLY=1; shift ;;
    --no-restart)    NO_RESTART=1; shift ;;
    --force-config)  FORCE_CONFIG=1; shift ;;
    --force-unit)    FORCE_UNIT=1; shift ;;
    --dry-run)       DRY_RUN=1; shift ;;
    -y|--yes)        ASSUME_YES=1; shift ;;
    -h|--help)       usage; exit 0 ;;
    *) die "未知参数：$1" "用 -h 查看用法" ;;
  esac
done
if [ "$NO_RESTART" = 1 ]; then RESTART_MODE="none"; fi
if [ "$PACKAGE_ONLY" != 1 ] && [ -z "$REMOTE_HOST" ]; then
  die "没有服务器地址" "填 $CONFIG_FILE 里的 REMOTE_HOST，或用 -H 1.2.3.4" \
      "模板：cp deploy/deploy.env.example deploy/deploy.env"
fi

# ---------- SSH（连接复用，避免重复认证）----------
CTRL_DIR=""; STAGE_DIR=""
cleanup() {
  if [ -n "$CTRL_DIR" ]; then
    [ -S "$CTRL_DIR/cm" ] && ssh -o "ControlPath=$CTRL_DIR/cm" -O exit x >/dev/null 2>&1
    rm -rf "$CTRL_DIR"
  fi
  [ -n "$STAGE_DIR" ] && rm -rf "$STAGE_DIR"
  return 0
}
trap cleanup EXIT

SSH_PRE=(); SSH_OPTS=()
setup_ssh() {
  CTRL_DIR="$(mktemp -d "${TMPDIR:-/tmp}/sub2api-deploy.XXXXXX")"
  SSH_OPTS=(-o ConnectTimeout=10 -o "StrictHostKeyChecking=$SSH_STRICT"
            -o ServerAliveInterval=30 -o ControlMaster=auto
            -o "ControlPath=$CTRL_DIR/cm" -o ControlPersist=180)
  if [ -n "$SSH_KEY" ]; then
    SSH_KEY="${SSH_KEY/#\~/$HOME}"
    [ -f "$SSH_KEY" ] || die "SSH 私钥不存在：$SSH_KEY" "先跑：deploy/deploy-remote.sh --setup-key"
    SSH_OPTS+=(-i "$SSH_KEY" -o IdentitiesOnly=yes)
  fi
  if [ -n "$SSH_PASSWORD" ]; then
    command -v sshpass >/dev/null 2>&1 \
      || die "配了 SSH_PASSWORD 但没装 sshpass" "sudo apt install sshpass" \
              "更推荐改用密钥免密：deploy/deploy-remote.sh --setup-key"
    SSH_PRE=(sshpass -p "$SSH_PASSWORD")
    SSH_OPTS+=(-o PubkeyAuthentication=no -o PreferredAuthentications=password -o BatchMode=no)
  else
    SSH_OPTS+=(-o BatchMode=yes)
  fi
}
rssh() {
  "${SSH_PRE[@]:+${SSH_PRE[@]}}" ssh "${SSH_OPTS[@]}" -p "$REMOTE_PORT" \
    "$REMOTE_USER@$REMOTE_HOST" "$@"
}
rscp() {
  "${SSH_PRE[@]:+${SSH_PRE[@]}}" scp "${SSH_OPTS[@]}" -P "$REMOTE_PORT" \
    "$1" "$REMOTE_USER@$REMOTE_HOST:$2"
}

# ---------- --setup-key：把公钥装到服务器，之后永久免密 ----------
setup_key() {
  local key="${SSH_KEY:-$HOME/.ssh/id_ed25519}"
  key="${key/#\~/$HOME}"
  if [ ! -f "$key" ]; then
    info "本机没有 $key，生成新的 ed25519 密钥"
    ssh-keygen -t ed25519 -N '' -C "sub2api-deploy" -f "$key"
  fi
  info "安装公钥到 $REMOTE_USER@$REMOTE_HOST:$REMOTE_PORT"
  if [ -n "$SSH_PASSWORD" ]; then
    command -v sshpass >/dev/null 2>&1 || die "需要 sshpass 才能用密码自动装公钥"
    sshpass -p "$SSH_PASSWORD" ssh-copy-id -i "$key.pub" -p "$REMOTE_PORT" \
      -o "StrictHostKeyChecking=$SSH_STRICT" "$REMOTE_USER@$REMOTE_HOST"
  else
    warn "接下来需要输入一次服务器密码（仅此一次）"
    ssh-copy-id -i "$key.pub" -p "$REMOTE_PORT" \
      -o "StrictHostKeyChecking=$SSH_STRICT" "$REMOTE_USER@$REMOTE_HOST"
  fi
  SSH_KEY="$key"; SSH_PASSWORD=""
  setup_ssh
  rssh true || die "公钥装好了但免密登录仍失败" "检查服务器 sshd 的 PubkeyAuthentication 配置"
  ok "免密登录已就绪，密钥：$key"
  ok "把 SSH_KEY=\"$key\" 写进 $CONFIG_FILE 即可"
}
# ---------- 本地构建 ----------
GO_BIN=""
resolve_go() {
  if command -v go >/dev/null 2>&1; then GO_BIN="$(command -v go)"; return 0; fi
  local c
  for c in /usr/local/go/bin/go "$HOME/.local/go/bin/go" "$HOME/go/bin/go" /opt/go/bin/go; do
    if [ -x "$c" ]; then GO_BIN="$c"; return 0; fi
  done
  die "找不到 go（backend/go.mod 要求 1.26.5）" "安装后重试，或先把 go 加进 PATH"
}

build_frontend() {
  command -v pnpm >/dev/null 2>&1 || die "找不到 pnpm（本仓库前端只用 pnpm）" "npm i -g pnpm"
  info "构建前端 → backend/internal/web/dist"
  pnpm --dir "$REPO_DIR/frontend" install
  pnpm --dir "$REPO_DIR/frontend" run build
}

build_backend() {
  resolve_go
  info "构建后端 linux/$TARGET_ARCH（-tags embed，内嵌前端）"
  (
    cd "$REPO_DIR/backend"
    CGO_ENABLED=0 GOOS=linux GOARCH="$TARGET_ARCH" "$GO_BIN" build \
      -tags embed -trimpath \
      -ldflags "-s -w -X main.Version=$VERSION -X main.Commit=$GIT_COMMIT -X main.Date=$BUILD_DATE" \
      -o "$BIN_OUT" ./cmd/server
  )
}

VERSION="$("$REPO_DIR/backend/scripts/resolve-version.sh" 2>/dev/null || echo "0.0.0-dev")"
GIT_COMMIT="$(git -C "$REPO_DIR" rev-parse --short HEAD 2>/dev/null || echo unknown)"
BUILD_DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
STAMP="$(date +%Y%m%d-%H%M%S)"
BIN_OUT="$REPO_DIR/sub2api"
TARBALL="$REPO_DIR/dist/sub2api-$VERSION-linux-$TARGET_ARCH-$STAMP.tar.gz"
REMOTE_TMP="/tmp/sub2api-deploy-$STAMP"

printf '%s\n' "----------------------------------------------"
printf '  版本     : %s (%s)\n' "$VERSION" "$GIT_COMMIT"
printf '  目标架构 : linux/%s\n' "$TARGET_ARCH"
if [ "$PACKAGE_ONLY" = 1 ]; then
  printf '  模式     : 仅本地打包\n'
else
  printf '  服务器   : %s@%s:%s\n' "$REMOTE_USER" "$REMOTE_HOST" "$REMOTE_PORT"
  printf '  远端目录 : %s\n' "$REMOTE_DIR"
  printf '  服务     : %s（重启方式 %s，健康检查 :%s/health）\n' \
    "$SERVICE_NAME" "$RESTART_MODE" "$APP_PORT"
  if [ -n "$SSH_PASSWORD" ]; then
    printf '  免密方式 : sshpass 密码\n'
  else
    printf '  免密方式 : SSH 密钥 %s\n' "${SSH_KEY:-ssh-agent/默认key}"
  fi
fi
printf '%s\n' "----------------------------------------------"

if [ "$SETUP_KEY" = 1 ]; then
  setup_key
  exit 0
fi

if [ "$DRY_RUN" = 1 ]; then
  info "dry-run，只列出将要执行的步骤："
  [ "$SKIP_FRONTEND" = 1 ] && printf '  - 跳过前端构建\n' || printf '  - pnpm install && pnpm run build\n'
  [ "$SKIP_BUILD" = 1 ] && printf '  - 跳过后端构建，直接用 %s\n' "$BIN_OUT" \
    || printf '  - go build -tags embed → %s\n' "$BIN_OUT"
  printf '  - 打包 → %s\n' "$TARBALL"
  if [ "$PACKAGE_ONLY" != 1 ]; then
    printf '  - scp → %s:%s/pkg.tar.gz\n' "$REMOTE_HOST" "$REMOTE_TMP"
    printf '  - ssh 执行 remote-install.sh（换二进制 → 重启 %s → /health 校验 → 失败回滚）\n' "$SERVICE_NAME"
  fi
  exit 0
fi

# ---------- 发布前：先确认能免密连上，再花时间构建 ----------
if [ "$PACKAGE_ONLY" != 1 ]; then
  setup_ssh
  info "检查免密 SSH 连通性"
  rssh true 2>/dev/null || die "免密 SSH 连接失败：$REMOTE_USER@$REMOTE_HOST:$REMOTE_PORT" \
    "先配免密：deploy/deploy-remote.sh --setup-key" \
    "或在 $CONFIG_FILE 里设 SSH_KEY / SSH_PASSWORD"
  SUDO_STATE="$(rssh 'if [ "$(id -u)" -eq 0 ]; then echo root; elif sudo -n true 2>/dev/null; then echo sudo; else echo nosudo; fi')"
  if [ "$SUDO_STATE" = "nosudo" ]; then
    die "远端用户 $REMOTE_USER 没有免密 sudo，无法重启服务" \
      "在服务器上执行（一次即可）：" \
      "  echo '$REMOTE_USER ALL=(ALL) NOPASSWD:ALL' | sudo tee /etc/sudoers.d/sub2api-deploy" \
      "或把 REMOTE_USER 改成 root"
  fi
  ok "免密通道就绪（远端权限：$SUDO_STATE）"

  if [ "$ASSUME_YES" != 1 ]; then
    if [ -t 0 ]; then
      read -r -p "确认发布到 $REMOTE_USER@$REMOTE_HOST:$REMOTE_DIR ? [y/N] " ans
      case "$ans" in
        y|Y|yes|YES) ;;
        *) die "已取消" ;;
      esac
    else
      die "非交互环境请加 -y 确认"
    fi
  fi
fi

# ---------- 构建 ----------
if [ "$SKIP_BUILD" = 1 ]; then
  [ -s "$BIN_OUT" ] || die "--skip-build 但 $BIN_OUT 不存在"
  warn "跳过构建，直接发布现有 $BIN_OUT"
else
  if [ "$SKIP_FRONTEND" = 1 ]; then
    [ -f "$REPO_DIR/backend/internal/web/dist/index.html" ] \
      || die "--skip-frontend 但 backend/internal/web/dist 里没有 index.html"
    warn "跳过前端构建，复用已有 dist"
  else
    build_frontend
  fi
  build_backend
fi
[ -s "$BIN_OUT" ] || die "构建产物为空：$BIN_OUT"
ok "二进制就绪：$BIN_OUT（$(du -h "$BIN_OUT" | cut -f1)）"
# ---------- 打包（二进制 + 配置模板 + unit + 远端安装脚本 + 参数）----------
kv() { printf "%s='%s'\n" "$1" "$(printf '%s' "$2" | sed "s/'/'\\\\''/g")"; }

info "打包"
mkdir -p "$REPO_DIR/dist"
STAGE_DIR="$(mktemp -d "${TMPDIR:-/tmp}/sub2api-stage.XXXXXX")"
cp "$BIN_OUT" "$STAGE_DIR/sub2api"
cp "$SCRIPT_DIR/config.personal.sqlite.yaml" "$STAGE_DIR/"
cp "$SCRIPT_DIR/sub2api-sqlite.service" "$STAGE_DIR/"
cp "$SCRIPT_DIR/remote-install.sh" "$STAGE_DIR/"
{
  echo "# 由 deploy/deploy-remote.sh 生成，随包上传"
  kv APP_DIR "$REMOTE_DIR"
  kv SERVICE_NAME "$SERVICE_NAME"
  kv SERVICE_USER "$SERVICE_USER"
  kv APP_PORT "$APP_PORT"
  kv RESTART_MODE "$RESTART_MODE"
  kv RESTART_COMMAND "$RESTART_COMMAND"
  kv HEALTH_TIMEOUT "$HEALTH_TIMEOUT"
  kv KEEP_RELEASES "$KEEP_RELEASES"
  kv FORCE_CONFIG "$FORCE_CONFIG"
  kv FORCE_UNIT "$FORCE_UNIT"
  kv ADMIN_EMAIL "$ADMIN_EMAIL"
  kv ADMIN_PASSWORD "$ADMIN_PASSWORD"
  kv RELEASE_STAMP "$STAMP"
  kv APP_VERSION "$VERSION"
} >"$STAGE_DIR/deploy-vars.env"
chmod 600 "$STAGE_DIR/deploy-vars.env"
tar -czf "$TARBALL" -C "$STAGE_DIR" .
ok "包已生成：$TARBALL（$(du -h "$TARBALL" | cut -f1)）"

if [ "$PACKAGE_ONLY" = 1 ]; then
  info "仅打包模式：上传解压后在服务器执行 bash remote-install.sh 即可"
  exit 0
fi
# ---------- 上传 + 远端安装 ----------
info "上传到 $REMOTE_HOST:$REMOTE_TMP"
rssh "mkdir -p -m 700 '$REMOTE_TMP'"
rscp "$TARBALL" "$REMOTE_TMP/pkg.tar.gz"
rssh "tar -xzf '$REMOTE_TMP/pkg.tar.gz' -C '$REMOTE_TMP'"
ok "上传完成"

info "服务器上安装并重启"
DEPLOY_RC=0
rssh "bash '$REMOTE_TMP/remote-install.sh'" || DEPLOY_RC=$?
rssh "rm -rf '$REMOTE_TMP'" || true

if [ "$DEPLOY_RC" -ne 0 ]; then
  die "远端安装失败（脚本已尝试自动回滚），退出码 $DEPLOY_RC" \
    "查日志：ssh -p $REMOTE_PORT $REMOTE_USER@$REMOTE_HOST \"journalctl -u $SERVICE_NAME -n 100 --no-pager\""
fi

ok "发布完成"
printf '  版本   : %s (%s)\n' "$VERSION" "$GIT_COMMIT"
printf '  访问   : http://%s:%s\n' "$REMOTE_HOST" "$APP_PORT"
printf '  日志   : ssh -p %s %s@%s "journalctl -u %s -f"\n' \
  "$REMOTE_PORT" "$REMOTE_USER" "$REMOTE_HOST" "$SERVICE_NAME"
printf '  回滚   : ssh -p %s %s@%s "sudo install -m0755 %s/sub2api.prev %s/sub2api && sudo systemctl restart %s"\n' \
  "$REMOTE_PORT" "$REMOTE_USER" "$REMOTE_HOST" "$REMOTE_DIR" "$REMOTE_DIR" "$SERVICE_NAME"
printf '  产物   : %s\n' "$TARBALL"





