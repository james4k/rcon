GO_FLAGS = -ldflags "-X 'github.com/leighmacdonald/rcon/rcon.BuildVersion=`git describe --abbrev=0`'"
all:
	@go build $(GO_FLAGS) -o build/rcon main.go

win:
	@GOOS=windows GOARCH=amd64 go build $(GO_FLAGS) -o build/rcon.exe main.go

dist:
	@zip -j rcon-`git describe --abbrev=0`-win64.zip build/rcon.exe LICENSE
	@zip -j rcon-`git describe --abbrev=0`-linux64.zip build/rcon LICENSE