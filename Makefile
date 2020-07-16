all:
	@go build -o build/rcon main.go

win:
	@GOOS=windows GOARCH=amd64 go build -o build/rcon.exe main.go

dist:
	@zip -j rcon-`git describe --abbrev=0`-win64.zip build/rcon.exe LICENSE
	@zip -j rcon-`git describe --abbrev=0`-linux64.zip build/rcon LICENSE