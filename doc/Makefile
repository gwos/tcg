# This Makefile provides a means to test formatting of markdown files locally in a
# terminal window.  External tooling like pandoc may be required.

# The build instructions here are presently crude and subject to change.

all	: readme

readme	:
	# pandoc README.gotocjson.md | lynx -stdin
	# pandoc --from markdown-smart -t plain README.gotocjson.md | more
	# pandoc -s -f markdown -t man README.gotocjson.md | man -l -
	# pandoc --from markdown-smart -t plain README.gotocjson.md | more
	# pandoc --from markdown_github --to plain --standalone README.gotocjson.md | more
	pandoc --from gfm --to plain --standalone README.gotocjson.md | more

# jq --slurp --raw-input '{"text": "\(.)", "mode": "markdown"}' < README.gotocjson.md | curl    --data @- https://api.github.com/markdown > README.gotocjson.html
# jq --slurp --raw-input '{"text": "\(.)", "mode": "markdown"}' < README.gotocjson.md | curl -s --data @- https://api.github.com/markdown > README.gotocjson.html
