//go:generate sh -c "curl -fsSL https://raw.githubusercontent.com/LadybugDB/ladybug/refs/heads/main/scripts/download-liblbug.sh | LBUG_TARGET_DIR=../../lib-ladybug bash"
//go:generate sh -c "[ \"$(uname)\" = Darwin ] && ln -sf liblbug.dylib ../../lib-ladybug/liblbug.0.dylib || ln -sf liblbug.so ../../lib-ladybug/liblbug.so.0"
package utils
