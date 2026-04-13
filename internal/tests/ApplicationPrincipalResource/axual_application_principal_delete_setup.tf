resource "axual_environment" "tf-test-env-delete" {
  name                 = "tf-development-delete"
  short_name           = "tfdevdel"
  description          = "This is the terraform testing environment for delete tests"
  color                = "#19b9be"
  visibility           = "Private"
  authorization_issuer = "Auto"
  instance             = data.axual_instance.test_instance.id
  owners               = data.axual_group.test_group.id
}

resource "axual_application" "tf-test-app-delete" {
  name             = "tf-test-app-delete"
  application_type = "Custom"
  short_name       = "tf_test_app_del"
  application_id   = "tf.test.app.delete"
  owners           = data.axual_group.test_group.id
  type             = "Java"
  visibility       = "Public"
  description      = "Axual's TF Test Application for delete tests"
}
