.PHONY: mock
mock:
	mockery --dir=services --name=UserService --output=mocks
	mockery --dir=services --name=EmailService --output=mocksma