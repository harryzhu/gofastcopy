rm -f /Volumes/harry/dev/app/py/t5/gofastcopy
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o dist/macos_arm/gofastcopy -ldflags "-w -s" main.go
cp dist/macos_arm/gofastcopy /Volumes/harry/dev/app/py/t5/
chmod +x /Volumes/harry/dev/app/py/t5/gofastcopy
#zip dist/macos_arm/filewalk_macos_arm.zip dist/macos_arm/gofastcopy

#CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o dist/macos_intel/gofastcopy -ldflags "-w -s" main.go
#zip dist/macos_intel/filewalk_macos_intel.zip dist/macos_intel/gofastcopy


# CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o dist/linux_amd64/gofastcopy -ldflags "-w -s" main.go
# zip dist/linux_amd64/filewalk_linux_amd64.zip dist/linux_amd64/gofastcopy


#CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o dist/windows_amd64/gofastcopy.exe -ldflags "-w -s" main.go
#zip dist/windows_amd64/filewalk_windows_amd64.zip dist/windows_amd64/gofastcopy.exe
