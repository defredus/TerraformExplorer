variable "valid" {
  default = "still parse the file"
}

resource "aws_s3_bucket" "broken" {
  bucket =
}
