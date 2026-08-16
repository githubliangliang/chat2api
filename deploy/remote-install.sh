#!/usr/bin/env bash
#
# 在 VPS 上执行：装新二进制 → 重启服务 → 健康检查 → 失败自动回滚。
# 由 deploy/deploy-remote.sh 打包上传后调用，一般不需要手动跑。
# 参数来自同目录的 deploy-vars.env。
#
set -Eeuo pipefail

HERE="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
[ -f "$HERE/deploy-vars.env" ] || { echo "缺少 deploy-vars.env" >&2; exit 1; }
# shellcheck source=/dev/null
. "$HERE/deploy-vars.env"

: "${APP_DIR:=/opt/sub2api}"
: "${SERVICE_NAME:=sub2api}"
: "${SERVICE_USER:=sub2api}"
: "${APP_PORT:=8080}"
: "${RESTART_MODE:=systemd}"
: "${RESTART_COMMAND:=}"
: "${HEALTH_TIMEOUT:=60}"
: "${KEEP_RELEASES:=3}"
: "${FORCE_CONFIG:=0}"
: "${FORCE_UNIT:=0}"
: "${ADMIN_EMAIL:=}"
: "${ADMIN_PASSWORD:=}"
: "${RELEASE_STAMP:=$(date +%Y%m%d-%H%M%S)}"

log()  { printf '\033[0;36m[remote]\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m[remote]\033[0m %s\n' "$*" >&2; }
die()  { printf '\033[0;31m[remote] %s\033[0m\n' "$*" >&2; exit 1; }

# root 直接执行，普通用户走免密 sudo
sudo_run() {
  if [ "$(id -u)" -eq 0 ]; then "$@"; else sudo -n "$@"; fi
}

DATA_DIR="$APP_DIR/data"
APP_BIN="$APP_DIR/sub2api"
PREV_BIN="$APP_DIR/sub2api.prev"
REL_DIR="$APP_DIR/releases/$RELEASE_STAMP"
UNIT_FILE="/etc/systemd/system/$SERVICE_NAME.service"
CONF_FILE="$DATA_DIR/config.yaml"
NEW_CONFIG=0

# ---------- 1. 前置检查：先确认新二进制能在这台机器上跑，再动线上服务 ----------
[ -f "$HERE/sub2api" ] || die "包里没有 sub2api 二进制"
if [ "$(id -u)" -ne 0 ] && ! sudo -n true 2>/dev/null; then
  die "当前用户没有免密 sudo，无法安装/重启服务"
fi
chmod +x "$HERE/sub2api"
if ! VER_LINE="$("$HERE/sub2api" -version 2>&1 | head -1)"; then
  die "新二进制无法执行（架构可能不匹配）：$VER_LINE"
fi
log "新版本自检通过：$VER_LINE"

# ---------- 2. 目录 / 运行用户 ----------
sudo_run mkdir -p "$DATA_DIR" "$REL_DIR"

if [ "$SERVICE_USER" != "root" ] && ! id -u "$SERVICE_USER" >/dev/null 2>&1; then
  log "创建运行用户 $SERVICE_USER"
  NOLOGIN=/usr/sbin/nologin
  [ -x "$NOLOGIN" ] || NOLOGIN=/sbin/nologin
  [ -x "$NOLOGIN" ] || NOLOGIN=/bin/false
  sudo_run useradd --system --no-create-home --shell "$NOLOGIN" "$SERVICE_USER" \
    || warn "创建用户失败，沿用现有配置"
fi

sudo_run install -m 0755 "$HERE/sub2api" "$REL_DIR/sub2api"

# ---------- 3. 配置：已有 config.yaml 绝不覆盖（除非 FORCE_CONFIG=1）----------
if [ -f "$CONF_FILE" ] && [ "$FORCE_CONFIG" != "1" ]; then
  log "保留已有 config.yaml"
else
  if [ -f "$CONF_FILE" ]; then
    sudo_run cp -a "$CONF_FILE" "$CONF_FILE.bak.$RELEASE_STAMP"
    warn "已覆盖 config.yaml，旧文件备份为 config.yaml.bak.$RELEASE_STAMP"
  fi
  TMP_CONF="$(mktemp)"
  sed -e "s#/opt/sub2api#$APP_DIR#g" "$HERE/config.personal.sqlite.yaml" >"$TMP_CONF"
  JWT_SECRET="$(openssl rand -hex 32 2>/dev/null || od -An -tx1 -N32 /dev/urandom | tr -d ' \n')"
  # TOTP 密钥必须固定：留空由程序每次启动随机生成，重启即作废已绑定的 2FA
  TOTP_KEY="$(openssl rand -hex 32 2>/dev/null || od -An -tx1 -N32 /dev/urandom | tr -d ' \n')"
  if [ -z "$ADMIN_PASSWORD" ]; then
    ADMIN_PASSWORD="$(openssl rand -hex 9 2>/dev/null || od -An -tx1 -N9 /dev/urandom | tr -d ' \n')"
  fi
  sed -i -e "s#CHANGE_ME_openssl_rand_hex_32#$JWT_SECRET#" \
         -e "s#CHANGE_ME_totp_hex_32#$TOTP_KEY#" \
         -e "s#CHANGE_ME_STRONG_PASSWORD#$ADMIN_PASSWORD#" "$TMP_CONF"
  if [ -n "$ADMIN_EMAIL" ]; then
    sed -i -e "s#admin@example.com#$ADMIN_EMAIL#" "$TMP_CONF"
  fi
  sudo_run install -m 0640 "$TMP_CONF" "$CONF_FILE"
  rm -f "$TMP_CONF"
  NEW_CONFIG=1
  log "已生成 $CONF_FILE（jwt.secret / totp.encryption_key 随机）"
  log "初始管理员：${ADMIN_EMAIL:-admin@example.com} / $ADMIN_PASSWORD"
fi
# ---------- 4. systemd unit：缺失才写，已有的不动（除非 FORCE_UNIT=1）----------
if [ "$RESTART_MODE" = "systemd" ]; then
  if [ -f "$UNIT_FILE" ] && [ "$FORCE_UNIT" != "1" ]; then
    log "保留已有 $UNIT_FILE"
  else
    [ -f "$HERE/sub2api-sqlite.service" ] || die "包里缺少 systemd unit 模板"
    TMP_UNIT="$(mktemp)"
    sed -e "s#/opt/sub2api#$APP_DIR#g" \
        -e "s#^User=.*#User=$SERVICE_USER#" \
        -e "s#^Group=.*#Group=$SERVICE_USER#" \
        -e "s#^Environment=SERVER_PORT=.*#Environment=SERVER_PORT=$APP_PORT#" \
        "$HERE/sub2api-sqlite.service" >"$TMP_UNIT"
    sudo_run install -m 0644 "$TMP_UNIT" "$UNIT_FILE"
    rm -f "$TMP_UNIT"
    sudo_run systemctl daemon-reload
    log "已安装 $UNIT_FILE"
  fi
fi

# ---------- 5. 换二进制：备份旧的 + 原子 rename（运行中也能替换）----------
if [ -f "$APP_BIN" ]; then
  sudo_run cp -a "$APP_BIN" "$PREV_BIN"
  log "旧二进制已备份 → $PREV_BIN"
fi
sudo_run install -m 0755 "$REL_DIR/sub2api" "$APP_BIN.new"
sudo_run mv -f "$APP_BIN.new" "$APP_BIN"

if [ "$SERVICE_USER" != "root" ] && id -u "$SERVICE_USER" >/dev/null 2>&1; then
  sudo_run chown -R "$SERVICE_USER:$SERVICE_USER" "$APP_DIR"
fi

# ---------- 6. 重启 ----------
restart_service() {
  case "$RESTART_MODE" in
    systemd)
      sudo_run systemctl daemon-reload
      sudo_run systemctl enable "$SERVICE_NAME" >/dev/null 2>&1 || true
      sudo_run systemctl restart "$SERVICE_NAME"
      log "已重启 systemd 服务 $SERVICE_NAME"
      ;;
    command)
      [ -n "$RESTART_COMMAND" ] || die "RESTART_MODE=command 但 RESTART_COMMAND 为空"
      log "执行自定义重启命令：$RESTART_COMMAND"
      bash -c "$RESTART_COMMAND"
      ;;
    none) log "RESTART_MODE=none，只换二进制不重启" ;;
    *) die "未知 RESTART_MODE=$RESTART_MODE" ;;
  esac
}
restart_service
# ---------- 7. 健康检查，失败回滚 ----------
health_probe() {
  if command -v curl >/dev/null 2>&1; then
    curl -fsS --max-time 3 "http://127.0.0.1:$APP_PORT/health" >/dev/null 2>&1
  elif command -v wget >/dev/null 2>&1; then
    wget -q -T 3 -O /dev/null "http://127.0.0.1:$APP_PORT/health" 2>/dev/null
  else
    return 2
  fi
}

rollback() {
  warn "健康检查失败（${HEALTH_TIMEOUT}s 内 /health 未就绪）"
  if [ -f "$PREV_BIN" ]; then
    sudo_run install -m 0755 "$PREV_BIN" "$APP_BIN.new"
    sudo_run mv -f "$APP_BIN.new" "$APP_BIN"
    restart_service || true
    warn "已回滚到上一版本，请查日志排查后重试"
  else
    warn "没有上一版本可回滚（首次部署？）"
  fi
  if [ "$RESTART_MODE" = "systemd" ]; then
    sudo_run journalctl -u "$SERVICE_NAME" -n 40 --no-pager || true
  fi
  exit 1
}

if [ "$RESTART_MODE" = "none" ]; then
  log "跳过健康检查"
else
  PROBE_RC=0
  health_probe || PROBE_RC=$?
  if [ "$PROBE_RC" -eq 2 ]; then
    warn "服务器没有 curl/wget，跳过健康检查"
  else
    log "等待 /health 就绪（最多 ${HEALTH_TIMEOUT}s）"
    WAITED=0
    until health_probe; do
      if [ "$WAITED" -ge "$HEALTH_TIMEOUT" ]; then rollback; fi
      sleep 2
      WAITED=$((WAITED + 2))
    done
    log "健康检查通过：http://127.0.0.1:$APP_PORT/health"
  fi
fi
# ---------- 8. 清理历史版本 ----------
if [ -d "$APP_DIR/releases" ] && [ "$KEEP_RELEASES" -gt 0 ]; then
  OLD_RELEASES="$(cd "$APP_DIR/releases" && ls -1 2>/dev/null | sort -r | tail -n "+$((KEEP_RELEASES + 1))")"
  if [ -n "$OLD_RELEASES" ]; then
    printf '%s\n' "$OLD_RELEASES" | while IFS= read -r d; do
      [ -n "$d" ] || continue
      sudo_run rm -rf -- "$APP_DIR/releases/$d"
    done
    log "已清理旧版本，保留最近 $KEEP_RELEASES 个"
  fi
fi

if [ "$RESTART_MODE" = "systemd" ]; then
  sudo_run systemctl status "$SERVICE_NAME" --no-pager --lines=0 || true
fi

log "部署完成：$VER_LINE"
if [ "$NEW_CONFIG" = "1" ]; then
  log "首次部署：管理员初始化见 deploy/START_NATIVE.md（AUTO_SETUP / -setup）"
fi
exit 0



