resource "axual_application_principal" "connector_delete_principal_1" {
  environment = axual_environment.tf-test-env.id
  application = axual_application.tf-test-app.id
  principal   = file("certs/generic_application_1.cer")
  private_key = file("certs/generic_application_1.key")
}
