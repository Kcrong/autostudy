# Not really..
lint:
	@pip install pylint
	pylint main.py


format:
	@pip install black
	black -l 79 main.py