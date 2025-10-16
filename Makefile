# Makefile
# NOTE: This makefile was intended to only work on my machine.
SHELL := pwsh.exe
.SHELLFLAGS := -NoProfile -Command

NAME = ShowAllFiles
BUILD_DIR = build
APP_FILE = $(BUILD_DIR)/$(NAME).exe

GOOS != $$(go env GOOS)
GOARCH != $$(go env GOARCH)
YEAR != $$((Get-Date).Year)
VERSION != $$(Get-Content VERSION -Raw)
FVERSION != $$(((Get-Content VERSION -Raw) -replace '\.',',') + ',0')

.PHONY: verify
verify:
	signtool.exe verify /v /pa $(APP_FILE)

.PHONY: resource
resource:
	@echo "`n-- Generating resources --"
	windres -D VERSION=$(VERSION) -D FVERSION=$(FVERSION) -D YEAR=$(YEAR) resource.rc -O coff -o resource.syso

.PHONY: sign
sign:
	@echo "`n-- Signing artifacts --"
	signtool.exe sign /v /n kamaran@layne.dev /tr http://timestamp.digicert.com /fd SHA256 /td SHA256 $(APP_FILE)

.PHONY: clean
clean:
	@echo "`n-- Cleaning old build artifacts --"
	Remove-Item $(BUILD_DIR)\* -Include "*.exe","*.zip","*.syso" -Force -ErrorAction SilentlyContinue; $$NULL = $$true

.PHONY: dist
dist: clean $(APP_FILE) $(BUILD_DIR)/setup.exe
	@echo "`n-- Archiving installer --"
	Compress-Archive $(BUILD_DIR)/setup.exe $(BUILD_DIR)/$(NAME)_v$(VERSION)_$(GOOS)_$(GOARCH).zip -CompressionLevel Optimal -Force

# installer
$(BUILD_DIR)/setup.exe: $(BUILD_DIR)/Root.crt $(BUILD_DIR)/CA.crt $(BUILD_DIR)/LICENSE.txt sign
	@echo "`n-- Compiling installer --"
	makensis /V1 /DVERSION=$(VERSION) /DYEAR=$(YEAR) /DBUILDDIR=$(BUILD_DIR) /DFILESIZE=$$((Get-Item $(APP_FILE)).Length/1KB) setup.nsi

# formatted license
$(BUILD_DIR)/LICENSE.txt:
	@echo "`n-- Generating formatted LICENSE --"
	(Get-Content LICENSE -Raw) -replace '(?<!\r?\n)\r?\n(?!\r?\n)',' ' -replace ' +',' ' -replace ' \(c\) '," `u{00A9} " | Out-File $@ -Encoding Unicode

# LayneRSARootCA
$(BUILD_DIR)/Root.crt:
	@echo "`n-- Downloading Root CA --"
	Invoke-RestMethod 'http://index.v.lan/etc/certs/rsa_root_ca.crt' | Out-File $@ -Encoding ASCII

# LayneRSACodeSigningCA
$(BUILD_DIR)/CA.crt:
	@echo "`n-- Downloading CodeSigning CA --"
	Invoke-RestMethod 'http://index.v.lan/etc/certs/rsa_codesign_ca.crt' | Out-File $@ -Encoding ASCII

# ShowAllFiles.exe
$(APP_FILE): resource
	@echo "`n-- Building $@ ($(GOOS)_$(GOARCH)) --"
	go build -v -tags=windows -ldflags "-H=windowsgui" -o $@
	@echo "# Last build: $$(Get-Date)n"
