GO_FLAGS = -ldflags "-X 'github.com/leighmacdonald/rcon/rcon.BuildVersion=`git describe --abbrev=0`'"

all: lin win mac

lin:
	@GOOS=linux GOARCH=amd64 go build $(GO_FLAGS) -o build/linux64/rcon main.go

win:
	@GOOS=windows GOARCH=amd64 go build $(GO_FLAGS) -o build/win64/rcon.exe main.go

mac:
	@GOOS=darwin GOARCH=amd64 go build $(GO_FLAGS) -o build/macos64/rcon main.go

dist:
	@zip -j rcon-`git describe --abbrev=0`-win64.zip build/win64/rcon.exe LICENSE
	@zip -j rcon-`git describe --abbrev=0`-linux64.zip build/linux64/rcon LICENSE
	@zip -j rcon-`git describe --abbrev=0`-macos64.zip build/macos64/rcon LICENSE
