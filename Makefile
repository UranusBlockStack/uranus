# Copyright 2016 The uranus Authors
# This file is part of the uranus library.
#
# The uranus library is free software: you can redistribute it and/or modify
# it under the terms of the GNU Lesser General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
#
# The uranus library is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
# GNU Lesser General Public License for more details.
#
# You should have received a copy of the GNU Lesser General Public License
# along with the uranus library. If not, see <http:#www.gnu.org/licenses/>.

TEST = $(shell go list ./... |grep -v test)
GOFILES_NOVENDOR := $(shell go list -f "{{.Dir}}" ./...)

### Check and format code 

# check the code for style standards; currently enforces go formatting.
# display output first, then check for success	
.PHONY: check
check:
	@echo "Checking code for formatting style compliance."
	@gofmt -l -d ${GOFILES_NOVENDOR}
	@gofmt -l ${GOFILES_NOVENDOR} | read && echo && echo "Your marmot has found a problem with the formatting style of the code." 1>&2 && exit 1 || true

# fmt runs gofmt -w on the code, modifying any files that do not match
# the style guide.
.PHONY: fmt
fmt:
	@echo "Correcting any formatting style corrections."
	@gofmt -l -w ${GOFILES_NOVENDOR}

### Test
.PHONY: test
test:
	@echo "Testing uranus all packages"
	@go test $(TEST)

### Building project

# build all targets 
.PHONY: check fmt all
all:
	@echo "Building all targets(uranus,uranuscli)."
	@go install ./cmd/uranus
	go build ./cmd/uranus
	mv uranus ./build

	@go install ./cmd/uranuscli
	go build ./cmd/uranuscli
	mv uranuscli ./build

# build all targets in windows 
.PHONY: win
win:
	@go install ./cmd/uranus
	go build ./cmd/uranus
	@move /y uranus.exe ./build >NUL

	@go install ./cmd/uranuscli
	go build ./cmd/uranuscli
	@move /y uranuscli.exe ./build >NUL
	@dir .\build\uranus*

### Clean up

# clean removes the target folder containing build artefacts
.PHONY: clean
clean:
	@echo "Clean uranus executable file."
	-rm ./build/uranus 
	-rm ./build/uranuscli

### Release
.PHONY: release
release: test 
	@echo "Making uranus release."
	@goreleaser --snapshot --rm-dist 

### solc
.PHONY: solc
solc: 
	@scripts/solc.sh ./build/solc
	@touch build/solc