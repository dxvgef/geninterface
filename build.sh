#!/usr/bin/env bash
set -e

# -----------------------
# 配置
# -----------------------
APP_NAME="geninterface"          # 可执行文件基础名
BUILD_DIR="build"       # 输出目录

GOOS_LIST=("linux" "windows" "darwin")
GOARCH_LIST=("amd64" "arm64")

# -----------------------
# 获取最新 Git 标签作为版本号
# -----------------------
VERSION=$(git describe --tags --abbrev=0 2>/dev/null || echo "dev")
echo "Building version: $VERSION"

# -----------------------
# 创建输出目录
# -----------------------
mkdir -p "$BUILD_DIR"

# -----------------------
# 循环构建并压缩
# -----------------------
for GOOS in "${GOOS_LIST[@]}"; do
    for GOARCH in "${GOARCH_LIST[@]}"; do
        BIN_NAME="${APP_NAME}_${GOOS}_${GOARCH}"
        [ "$GOOS" = "windows" ] && BIN_NAME="${BIN_NAME}.exe"

        echo "Building $BIN_NAME ..."

        # 禁用 Cgo 并优化编译
        CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH go build \
            -ldflags "-s -w -X 'main.Version=${VERSION}'" \
            -o "$BUILD_DIR/$BIN_NAME"

        echo "Built: $BUILD_DIR/$BIN_NAME"

        # 压缩
        PACKAGE_NAME="${BIN_NAME%.*}-${VERSION}"
        PACKAGE_PATH="$BUILD_DIR/$PACKAGE_NAME"

        if [ "$GOOS" = "windows" ]; then
            zip -j "${PACKAGE_PATH}.zip" "$BUILD_DIR/$BIN_NAME"
            echo "Compressed: ${PACKAGE_PATH}.zip"
        else
            tar -czf "${PACKAGE_PATH}.tar.gz" -C "$BUILD_DIR" "$BIN_NAME"
            echo "Compressed: ${PACKAGE_PATH}.tar.gz"
        fi

        # 删除原始二进制文件
        rm -f "$BUILD_DIR/$BIN_NAME"
    done
done

# -----------------------
# 生成 checksums.txt
# -----------------------
echo "Generating checksums.txt ..."
cd "$BUILD_DIR"
echo "# SHA256 checksums for version $VERSION" > checksums.txt
for f in $(ls *.{zip,tar.gz} 2>/dev/null | sort); do
    sha256sum "$f" | awk '{print $1 "  " $2}' >> checksums.txt
done
echo "Generated: $BUILD_DIR/checksums.txt"

echo "All builds, compressions, and checksums completed. Original binaries removed."
