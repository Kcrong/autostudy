# Not really..
lint:
	@pip install pylint
	pylint main.py


format:
	@pip install isort black
	isort main.py
	black -l 79 main.py

freeze:
	pip freeze > requirements.txt
