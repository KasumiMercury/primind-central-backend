data "external_schema" "gorm" {
  program = [
    "./scripts/gorm-schema.sh",
  ]
}

env "gorm" {
  src = data.external_schema.gorm.url
  dev = "docker://postgres/17"

  migration {
    dir = "file://migrations"
  }
}
