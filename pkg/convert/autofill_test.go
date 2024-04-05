package convert

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hexops/autogold/v2"
	"github.com/stretchr/testify/require"
)

type testAutoFiller struct {
	byToken map[string]string
}

var _ AutoFiller = (*testAutoFiller)(nil)

func (x *testAutoFiller) AutoFill(t, n string) string {
	return strings.ReplaceAll(x.byToken[t], `"example"`, fmt.Sprintf("\"%s\"", n))
}

func (x *testAutoFiller) CanAutoFill(t string) bool {
	_, ok := x.byToken[t]
	return ok
}

func TestAutoFill(t *testing.T) {
	example := `
resource "aws_route53_record" "example" {
      for_each = {
        for dvo in aws_acm_certificate.example.domain_validation_options : dvo.domain_name => {
          name   = dvo.resource_record_name
          record = dvo.resource_record_value
          type   = dvo.resource_record_type
        }
      }

      allow_overwrite = true
      name            = each.value.name
      records         = [each.value.record]
      ttl             = 60
      type            = each.value.type
      zone_id         = aws_route53_zone.example.zone_id
}`

	injectAcmCert := `
resource "aws_acm_certificate" "example" {
  domain_name       = "example.com"
  validation_method = "DNS"
}`

	injectRoute53Zone := `
resource "aws_route53_zone" "example" {
  name = "example.com"
}`

	taf := testAutoFiller{
		byToken: map[string]string{
			"aws_acm_certificate": injectAcmCert,
			"aws_route53_zone":    injectRoute53Zone,
		},
	}

	actual, err := AutoFill(&taf, example)
	require.NoError(t, err)

	t.Logf("ACTUAL: %s", actual)

	autogold.ExpectFile(t, actual)
}
