package alicloud

import (
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"

	"github.com/PaesslerAG/jsonpath"
	util "github.com/alibabacloud-go/tea-utils/service"
	"github.com/aliyun/terraform-provider-alicloud/alicloud/connectivity"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func resourceAlicloudDcdnKvNamespace() *schema.Resource {
	return &schema.Resource{
		Create: resourceAlicloudDcdnKvNamespaceCreate,
		Read:   resourceAlicloudDcdnKvNamespaceRead,
		Delete: resourceAlicloudDcdnKvNamespaceDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(5 * time.Minute),
			Delete: schema.DefaultTimeout(5 * time.Hour),
		},
		Schema: map[string]*schema.Schema{
			"description": {
				Required: true,
				ForceNew: true,
				Type:     schema.TypeString,
			},
			"namespace": {
				Required:     true,
				ForceNew:     true,
				Type:         schema.TypeString,
				ValidateFunc: validation.StringMatch(regexp.MustCompile(`[a-zA-Z0-9-_]+$`), "The name can contain letters, digits, hyphens (-), and underscores (_)."),
			},
			"status": {
				Computed: true,
				Type:     schema.TypeString,
			},
		},
	}
}

func resourceAlicloudDcdnKvNamespaceCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.AliyunClient)
	request := make(map[string]interface{})
	conn, err := client.NewDcdnClient()
	if err != nil {
		return WrapError(err)
	}

	if v, ok := d.GetOk("description"); ok {
		request["Description"] = v
	}
	if v, ok := d.GetOk("namespace"); ok {
		request["Namespace"] = v
	}

	var response map[string]interface{}
	action := "PutDcdnKvNamespace"
	wait := incrementalWait(3*time.Second, 3*time.Second)
	err = resource.Retry(client.GetRetryTimeout(d.Timeout(schema.TimeoutCreate)), func() *resource.RetryError {
		resp, err := conn.DoRequest(StringPointer(action), nil, StringPointer("POST"), StringPointer("2018-01-15"), StringPointer("AK"), nil, request, &util.RuntimeOptions{})
		if err != nil {
			if NeedRetry(err) {
				wait()
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		response = resp
		addDebug(action, response, request)
		return nil
	})
	if err != nil {
		return WrapErrorf(err, DefaultErrorMsg, "alicloud_dcdn_kv_namespace", action, AlibabaCloudSdkGoERROR)
	}

	if v, err := jsonpath.Get("$.Namespace", response); err != nil || v == nil {
		return WrapErrorf(err, IdMsg, "alicloud_dcdn_kv_namespace")
	} else {
		d.SetId(fmt.Sprint(v))
	}

	return resourceAlicloudDcdnKvNamespaceRead(d, meta)
}

func resourceAlicloudDcdnKvNamespaceRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.AliyunClient)
	dcdnService := DcdnService{client}

	object, err := dcdnService.DescribeDcdnKvNamespace(d.Id())
	if err != nil {
		if NotFoundError(err) {
			log.Printf("[DEBUG] Resource alicloud_dcdn_kv_namespace dcdnService.DescribeDcdnKvNamespace Failed!!! %s", err)
			d.SetId("")
			return nil
		}
		return WrapError(err)
	}
	d.Set("description", object["Description"])
	d.Set("namespace", object["Namespace"])
	d.Set("status", object["Status"])

	return nil
}

func resourceAlicloudDcdnKvNamespaceDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.AliyunClient)
	conn, err := client.NewDcdnClient()
	if err != nil {
		return WrapError(err)
	}
	dcdnService := DcdnService{client}

	request := map[string]interface{}{
		"Namespace": d.Id(),
	}

	action := "DeleteDcdnKvNamespace"
	wait := incrementalWait(3*time.Second, 3*time.Second)
	err = resource.Retry(client.GetRetryTimeout(d.Timeout(schema.TimeoutDelete)), func() *resource.RetryError {
		resp, err := conn.DoRequest(StringPointer(action), nil, StringPointer("POST"), StringPointer("2018-01-15"), StringPointer("AK"), nil, request, &util.RuntimeOptions{})
		if err != nil {
			if NeedRetry(err) {
				wait()
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		addDebug(action, resp, request)
		return nil
	})
	if err != nil {
		if NotFoundError(err) {
			return nil
		}
		return WrapErrorf(err, DefaultErrorMsg, d.Id(), action, AlibabaCloudSdkGoERROR)
	}
	stateConf := BuildStateConf([]string{}, []string{}, d.Timeout(schema.TimeoutDelete), 3*time.Second, dcdnService.DcdnKvNamespaceStateRefreshFunc(d.Id(), []string{}))
	if _, err := stateConf.WaitForState(); err != nil {
		return WrapErrorf(err, IdMsg, d.Id())
	}
	return nil
}
