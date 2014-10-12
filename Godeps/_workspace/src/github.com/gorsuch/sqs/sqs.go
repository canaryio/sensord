package sqs

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/url"

	"github.com/bmizerany/aws4"
)

type ErrorResponse struct {
	Error struct {
		Type    string
		Code    string
		Message string
	}
	RequestId string
}

type Message struct {
	MessageId     string
	ReceiptHandle string
	MD5OfBody     string
	Body          string
}

type ReceiveMessageResponse struct {
	Messages         []*Message `xml:"ReceiveMessageResult>Message"`
	ResponseMetadata struct {
		RequestId string
	}
}

func Get(queueURL, max string) ([]*Message, error) {
	v := url.Values{}
	v.Set("MaxNumberOfMessages", max)
	v.Set("Action", "ReceiveMessage")
	v.Set("WaitTimeSeconds", "20")

	res, err := aws4.PostForm(queueURL, v)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		var errorResponse ErrorResponse
		err := xml.Unmarshal(body, &errorResponse)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf(errorResponse.Error.Message)
	}

	var messageResponse ReceiveMessageResponse
	err = xml.Unmarshal(body, &messageResponse)
	if err != nil {
		return nil, err
	}
	return messageResponse.Messages, nil
}

func Delete(queueURL, handle string) error {
	v := url.Values{}
	v.Set("Action", "DeleteMessage")
	v.Set("ReceiptHandle", handle)

	res, err := aws4.PostForm(queueURL, v)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if res.StatusCode != 200 {
		var errorResponse ErrorResponse
		err := xml.Unmarshal(body, &errorResponse)
		if err != nil {
			return err
		}
		return fmt.Errorf(errorResponse.Error.Message)
	}

	return nil
}
