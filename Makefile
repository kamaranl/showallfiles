# Makefile
# NOTE: This makefile was intended to only work on my machine.
SHELL := pwsh.exe
.SHELLFLAGS := -NoProfile -Command

NAME = ShowAllFiles
ARCHS = 386 amd64 arm64
GOOS = windows
BUILD_DIR = build
APP_FILE = $(BUILD_DIR)/$(NAME).exe

YEAR != (Get-Date).Year
VERSION != Get-Content VERSION -Raw
COMMIT ?= $$(git rev-list --abbrev-commit -n1 HEAD)
DATE ?= $$(Get-Date -UFormat '%F')

# Shortcut for recursively building each target architecture defined above.
.PHONY: build
build: $(ARCHS)
	@echo "`n # Completed: $$(Get-Date)"

# Cleans the build directory of everything but certificates- since the data in
# the certificates does not change. It ends with $NULL = $true to force the
# recipe to end in success even if there were no files to remove.
.PHONY: clean
clean:
	@echo "`n-- Cleaning old build artifacts --"
	Remove-Item $(BUILD_DIR)\* -Exclude "*.crt" -Force -ErrorAction SilentlyContinue; $$NULL = $$true

# Downloads self-signed LayneRSARootCA & LayneRSACodeSigningCA.
certs:
	@echo "`n-- Downloading Root CA --"
	Invoke-RestMethod 'http://index.v.lan/etc/certs/rsa_root_ca.crt' | Out-File $(BUILD_DIR)/Root.crt -Encoding ASCII -ErrorAction SilentlyContinue -NoClobber; $$NULL = $$true
	@echo "`n-- Downloading CodeSigning CA --"
	Invoke-RestMethod 'http://index.v.lan/etc/certs/rsa_codesign_ca.crt' | Out-File $(BUILD_DIR)/CA.crt -Encoding ASCII -ErrorAction SilentlyContinue -NoClobber; $$NULL = $$true


# Makes a nicely formatted license to use in the installer.
license:
	@echo "`n-- Making formatted LICENSE --"
	(Get-Content LICENSE -Raw) -replace '(?<!\r?\n)\r?\n(?!\r?\n)',' ' -replace ' +',' ' -replace ' \(c\) '," `u{00A9} " | Out-File $(BUILD_DIR)/LICENSE.txt -Encoding Unicode -ErrorAction SilentlyContinue -NoClobber; $$NULL = $$true

# Makes the .syso files required- arm64 uses rsrc since windres doesn't have
# aarch64 as a target.
syso:
	@echo "`n-- Making .syso files --"
		rsrc -arch arm64 -ico internal/app/icons/$(NAME)1.ico -o $(BUILD_DIR)/resource_arm64.syso
		windres -D VERSION=$(VERSION) -D FVERSION=$$(((Get-Content VERSION -Raw) -replace '\.',',') + ',0') -D COPYYEAR=$(YEAR) -D BUILDDATE=$(DATE) -D BUILDCOMMIT=$(COMMIT) -i resource.rc -O coff -o $(BUILD_DIR)/resource_amd64.syso
		Copy-Item $(BUILD_DIR)/resource_amd64.syso $(BUILD_DIR)/resource_386.syso

# The meat and potatoes of the build & package sequence, it:
#   - Compiles the app
#   - Signs the app
#   - Compiles the installer
#   - Signs the installer
#   - Compresses the installer to a .zip archive
$(ARCHS): certs license syso
	@echo "`n-- Building $(NAME)_v$(VERSION)_$@.exe --"
	Copy-Item $(BUILD_DIR)/resource_$@.syso resource.syso -Force
	$$env:GOOS='$(GOOS)'; $$env:GOARCH='$@'; go build -v -tags=windows -ldflags "-H=windowsgui -X 'main.Version=$(VERSION) ($(COMMIT))'" -o $(BUILD_DIR)/$(NAME)_v$(VERSION)_$@.exe
	Copy-Item $(BUILD_DIR)/$(NAME)_v$(VERSION)_$@.exe $(APP_FILE) -Force

	@echo "`n-- Signing $(NAME).exe ($@) --"
	signtool.exe sign /n kamaran@layne.dev /tr http://timestamp.digicert.com /fd SHA256 /td SHA256 $(APP_FILE)

	@echo "`n-- Compiling setup.exe ($@) --"
	makensis /V1 /DVERSION=$(VERSION) /DBUILDDIR=$(BUILD_DIR) /DBUILDDATE=$(DATE) /DBUILDCOMMIT=$(COMMIT) /DFILESIZE=$$((Get-Item $(APP_FILE)).Length/1KB) setup.nsi; sleep 1.5

	@echo "`n-- Signing setup.exe ($@) --"
	signtool.exe sign /n kamaran@layne.dev /tr http://timestamp.digicert.com /fd SHA256 /td SHA256 $(BUILD_DIR)/setup.exe

	@echo "`n-- Compressing to $(BUILD_DIR)/$(NAME)_v$(VERSION)_$@-setup.zip --"
	Compress-Archive $(BUILD_DIR)/setup.exe $(BUILD_DIR)/$(NAME)_v$(VERSION)_$@-setup.zip -CompressionLevel Optimal -Force
