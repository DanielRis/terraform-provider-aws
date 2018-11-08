package aws

import (
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsSsmParameter() *schema.Resource {
	return &schema.Resource{
		Read: dataAwsSsmParameterRead,
		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"value": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
			"with_decryption": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"default": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "b7002342-3c99-4fec-8ef6-5b1bdcd00032",
			},
			"with_default": {
				Type:       schema.TypeBool,
				Optional:   true,
				Default:    false,
				Deprecated: "With default is deprecated, a default value is now returned if 'default' is set",
			},
			"walk_hierarchy": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
		},
	}
}

func dataAwsSsmParameterRead(d *schema.ResourceData, meta interface{}) error {
	ssmconn := meta.(*AWSClient).ssmconn

	name := d.Get("name").(string)

	paramInput := &ssm.GetParameterInput{
		Name:           aws.String(name),
		WithDecryption: aws.Bool(d.Get("with_decryption").(bool)),
	}

	log.Printf("[DEBUG] Reading SSM Parameter: %s", paramInput)
	resp, err := ssmconn.GetParameter(paramInput)

	if err != nil {
		if d.Get("walk_hierarchy").(bool) == true {
			split := strings.Split(name, "/")
			pathSlice := split[:len(split)-1]
			key := split[len(split)-1]
			for i := len(pathSlice); i > 0; i-- {
				ssmPath := append(pathSlice[0:i], key)
				paramInput = &ssm.GetParameterInput{
					Name:           aws.String(strings.Join(ssmPath, "/")),
					WithDecryption: aws.Bool(d.Get("with_decryption").(bool)),
				}
				log.Printf("[DEBUG] Reading SSM Parameter: %s", paramInput)
				resp, err := ssmconn.GetParameter(paramInput)
				if err != nil {
					if err.(awserr.Error).Code() == ssm.ErrCodeParameterNotFound {
						log.Println("[DEBUG] Parameter was not found but walk_hirarchy is enable. Moving down a level.")
					} else {
						return errwrap.Wrapf("[ERROR] Error describing SSM parameter: {{err}}", err)
					}
				} else {
					param := resp.Parameter
					d.SetId(*param.Name)

					arn := arn.ARN{
						Partition: meta.(*AWSClient).partition,
						Region:    meta.(*AWSClient).region,
						Service:   "ssm",
						AccountID: meta.(*AWSClient).accountid,
						Resource:  fmt.Sprintf("parameter/%s", strings.TrimPrefix(d.Id(), "/")),
					}
					d.Set("arn", arn.String())
					d.Set("name", param.Name)
					d.Set("type", param.Type)
					d.Set("value", param.Value)
					return nil
				}
			}
		}

		v, _ := d.GetOkExists("default")
		if err.(awserr.Error).Code() == ssm.ErrCodeParameterNotFound && (d.Get("default") != "b7002342-3c99-4fec-8ef6-5b1bdcd00032" || d.Get("with_default").(bool) == true) {
			if d.Get("default") == "b7002342-3c99-4fec-8ef6-5b1bdcd00032" {
				v = ""
			}
			d.SetId(name)
			d.Set("arn", "")
			d.Set("name", name)
			d.Set("type", "String")
			d.Set("value", v)
			return nil
		}

		return errwrap.Wrapf("[ERROR] Error describing SSM parameter: {{err}}", err)
	}

	param := resp.Parameter
	d.SetId(*param.Name)

	arn := arn.ARN{
		Partition: meta.(*AWSClient).partition,
		Region:    meta.(*AWSClient).region,
		Service:   "ssm",
		AccountID: meta.(*AWSClient).accountid,
		Resource:  fmt.Sprintf("parameter/%s", strings.TrimPrefix(d.Id(), "/")),
	}
	d.Set("arn", arn.String())
	d.Set("name", param.Name)
	d.Set("type", param.Type)
	d.Set("value", param.Value)

	return nil
}
