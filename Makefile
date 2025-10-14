# Makefile
SHELL := pwsh.exe
.SHELLFLAGS := -NoProfile -Command

GOOS = $$(go env GOOS)
GOARCH = $$(go env GOARCH)

NAME = ShowAllFiles
OUT_DIR = bin
OUT = $(OUT_DIR)/$(NAME).exe
VER = $$(Get-Content VERSION)

all: windows

windows: resources
	@echo "-- Building $(OUT) ($(GOOS)_$(GOARCH)) --"
	go build -v -tags=windows -ldflags "-H=windowsgui" -o $(OUT)
	@echo "# Last build: $$(Get-Date)"

resources:
	@echo "-- Generating resources --"
	go generate

signed:
	@echo "-- Signing artifacts --"
	signtool.exe sign /v /n kamaran@layne.dev /tr http://timestamp.digicert.com /fd SHA256 /td SHA256 $(OUT)

clean:
	@echo "-- Cleaning old build artifacts --"
	Remove-Item $(OUT) -Force
	Remove-Item resource.syso
	$(MAKE)

verify:
	signtool.exe verify /v /pa $(OUT)

bundle:
	Compress-Archive $(OUT) $(NAME)_v$(VER)_$(GOOS)_$(GOARCH).zip -CompressionLevel Optimal -Force

test:
	@echo "$(VER)"
