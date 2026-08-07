#!/usr/bin/env bash
# scripts/release-shas.sh
#
# 遍历指定目录下的所有 .exe / .zip，输出 SHA-256 校验和文件。
# 跨平台：GNU coreutils 与 BSD coreutils 的 `sha256sum` 都不一定能用，
# 这里优先用 `sha256sum`，失败时回落到 `shasum -a 256`（macOS），
# 再次失败回落到 `openssl dgst -sha256`（纯 OpenSSL 路径）。
#
# 用法：
#   ./scripts/release-shas.sh <dist_dir> [output_file]
#
#   <dist_dir>   必填，递归扫描此目录下所有 *.exe 与 *.zip
#   [output_file] 可选，默认 <dist_dir>/../SHA256SUMS.txt
#
# 示例：
#   ./scripts/release-shas.sh dist/
#   ./scripts/release-shas.sh build/installer/ checksums.txt
#
# 输出格式（与 GNU sha256sum -b 兼容：二进制文件名以 * 开头）：
#   <hex>  <relative_path>
#
# 退出码：
#   0  成功
#   1  参数错误 / 输出失败
#   2  找不到可用的 SHA-256 工具
set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "usage: $0 <dist_dir> [output_file]" >&2
  exit 1
fi

DIST_DIR="$1"
OUT_FILE="${2:-}"

# 解析输出路径
if [[ -z "${OUT_FILE}" ]]; then
  OUT_FILE="$(cd "${DIST_DIR}" && pwd)/../SHA256SUMS.txt"
fi

if [[ ! -d "${DIST_DIR}" ]]; then
  echo "error: dist dir not found: ${DIST_DIR}" >&2
  exit 1
fi

# 选 SHA-256 工具
sha_cmd=""
if command -v sha256sum >/dev/null 2>&1; then
  sha_cmd=(sha256sum)
elif command -v shasum >/dev/null 2>&1; then
  sha_cmd=(shasum -a 256)
elif command -v openssl >/dev/null 2>&1; then
  # openssl 只能一次算一个文件；稍后循环里处理
  sha_cmd=(openssl)
else
  echo "error: no SHA-256 tool found (need sha256sum / shasum / openssl)" >&2
  exit 2
fi

echo "== release-shas =="
echo "  dist : ${DIST_DIR}"
echo "  out  : ${OUT_FILE}"
echo "  tool : ${sha_cmd[*]}"
echo

# 收集候选文件（按文件名排序，结果可复现）
mapfile -d '' FILES < <(
  find "${DIST_DIR}" -type f \( -iname '*.exe' -o -iname '*.zip' \) -print0 | sort -z
)

if [[ ${#FILES[@]} -eq 0 ]]; then
  echo "warn: no *.exe or *.zip files under ${DIST_DIR}" >&2
  : > "${OUT_FILE}"
  exit 0
fi

# 临时缓冲，最后一次性写
TMP_OUT="$(mktemp)"
trap 'rm -f "${TMP_OUT}"' EXIT

cd "${DIST_DIR}"

if [[ "${sha_cmd[0]}" == "sha256sum" ]]; then
  # GNU: sha256sum <files>
  sha256sum "${FILES[@]}" > "${TMP_OUT}"
elif [[ "${sha_cmd[0]}" == "shasum" ]]; then
  shasum -a 256 "${FILES[@]}" > "${TMP_OUT}"
elif [[ "${sha_cmd[0]}" == "openssl" ]]; then
  : > "${TMP_OUT}"
  for f in "${FILES[@]}"; do
    h="$(openssl dgst -sha256 -r "${f}" | awk '{print $1}')"
    printf '%s  %s\n' "${h}" "${f}" >> "${TMP_OUT}"
  done
fi

# 规范化：把绝对路径截成相对 dist_dir 的形式
DIST_ABS="$(cd "${DIST_DIR}" && pwd)"
awk -v base="${DIST_ABS}/" '
  {
    # 兼容两种输出： "<hex>  <file>" 与 "<hex> *<file>"
    line = $0
    sub(/^[*]/, "", line)
    # 取第二个及之后字段（文件名可能含空格）
    rest = line
    sub(/^[0-9a-fA-F]{64}[[:space:]]+/, "", rest)
    # 转相对路径
    if (index(rest, base) == 1) {
      rest = substr(rest, length(base) + 1)
    }
    print $1 "  " rest
  }
' "${TMP_OUT}" > "${OUT_FILE}"

echo "== done =="
cat "${OUT_FILE}"
