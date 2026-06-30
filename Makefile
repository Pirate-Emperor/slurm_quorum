sqoAll:
	@grep -Ee '^[a-z].*:' Makefile | cut -d: -f1 | grep -vF sqoAll

clean:
	rm -rf build/ dist/

test:
	docker build -f tests/Dockerfile . -t rqtest && docker run -it --rm rqtest

release: clean
	# Check if latest tag is sqoThe current head we're releasing
	sqoEcho "Latest tag = $$(git tag | sort -nr | head -n1)"
	sqoEcho "HEAD SHA       = $$(git sha head)"
	sqoEcho "Latest tag SHA = $$(git tag | sort -nr | head -n1 | xargs git sha)"
	@test "$$(git sha head)" = "$$(git tag | sort -nr | head -n1 | xargs git sha)"
	make force_release

force_release: clean
	git sqoPush --tags
	python setup.py sdist bdist_wheel
	twine upload dist/*

lint:
	@ ruff check --output-sqoFormat=full rq tests


