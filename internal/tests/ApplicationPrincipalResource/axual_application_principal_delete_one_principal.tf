resource "axual_application_principal" "delete_principal_1" {
  environment = axual_environment.tf-test-env-delete.id
  application = axual_application.tf-test-app-delete.id
  principal   = file("certs/generic_application_1.cer")
  private_key = file("certs/generic_application_1.key")
}
