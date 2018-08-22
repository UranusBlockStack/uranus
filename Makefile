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

TEST = $(shell go list ./...)
all:
	@go install ./cmd/uranus
	go build ./cmd/uranus
	mv uranus ./build
run:
	@./build/uranus
stop:
clear:
test:
	go test $(TEST)

.PHONY: all run stop clear test
