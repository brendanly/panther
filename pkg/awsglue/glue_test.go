package awsglue

/**
 * Copyright 2020 Panther Labs Inc
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/glue"
	"github.com/aws/aws-sdk-go/service/glue/glueiface"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/panther-labs/panther/api/lambda/core/log_analysis/log_processor/models"
)

var (
	refTime             = time.Date(2020, 1, 3, 1, 1, 1, 0, time.UTC)
	nonAWSError         = errors.New("nonAWSError") // nolint:golint
	entityExistsError   = awserr.New(glue.ErrCodeAlreadyExistsException, "PartitionKey already exists.", nil)
	entityNotFoundError = awserr.New(glue.ErrCodeEntityNotFoundException, "Entity not found", nil)
	otherAWSError       = awserr.New("SomeException", "Some problem.", nil) // aws error other than those we code against

	testCreatePartitionOutput = &glue.CreatePartitionOutput{}

	testDeletePartitionOutput = &glue.DeletePartitionOutput{}

	testGetTableOutput = &glue.GetTableOutput{
		Table: &glue.TableData{
			StorageDescriptor: &glue.StorageDescriptor{
				Location: aws.String("s3://testbucket/logs/table"),
				SerdeInfo: &glue.SerDeInfo{
					SerializationLibrary: aws.String("org.openx.data.jsonserde.JsonSerDe"),
					Parameters: map[string]*string{
						"serialization.format": aws.String("1"),
						"case.insensitive":     aws.String("TRUE"),
					},
				},
			},
		},
	}
)

type partitionTestEvent struct{}

func TestGlueTableMetadataLogData(t *testing.T) {
	gm := NewGlueTableMetadata(models.LogData, "My.Logs.Type", "description", GlueTableHourly, partitionTestEvent{})

	assert.Equal(t, "description", gm.Description())
	assert.Equal(t, "My.Logs.Type", gm.LogType())
	assert.Equal(t, GlueTableHourly, gm.Timebin())
	assert.Equal(t, "my_logs_type", gm.TableName())
	assert.Equal(t, LogProcessingDatabaseName, gm.DatabaseName())
	assert.Equal(t, "logs/my_logs_type/", gm.Prefix())
	assert.Equal(t, partitionTestEvent{}, gm.eventStruct)
	assert.Equal(t, "logs/my_logs_type/year=2020/month=01/day=03/hour=01/", gm.GetPartitionPrefix(refTime))
}

func TestGlueTableMetadataRuleMatches(t *testing.T) {
	gm := NewGlueTableMetadata(models.RuleData, "My.Rule", "description", GlueTableHourly, partitionTestEvent{})

	assert.Equal(t, "description", gm.Description())
	assert.Equal(t, "My.Rule", gm.LogType())
	assert.Equal(t, GlueTableHourly, gm.Timebin())
	assert.Equal(t, "my_rule", gm.TableName())
	assert.Equal(t, RuleMatchDatabaseName, gm.DatabaseName())
	assert.Equal(t, "rules/my_rule/", gm.Prefix())
	assert.Equal(t, partitionTestEvent{}, gm.eventStruct)
	assert.Equal(t, "rules/my_rule/year=2020/month=01/day=03/hour=01/", gm.GetPartitionPrefix(refTime))
}

func TestCreateJSONPartition(t *testing.T) {
	gm := NewGlueTableMetadata(models.LogData, "Test.Logs", "Description", GlueTableHourly, partitionTestEvent{})

	// test no errors and partition does not exist (no error)
	glueClient := &mockGlue{}
	glueClient.On("GetTable", mock.Anything).Return(testGetTableOutput, nil).Once()
	glueClient.On("CreatePartition", mock.Anything).Return(testCreatePartitionOutput, nil).Once()
	assert.NoError(t, gm.CreateJSONPartition(glueClient, refTime))
	glueClient.AssertExpectations(t)
}

func TestCreateJSONPartitionPartitionExists(t *testing.T) {
	gm := NewGlueTableMetadata(models.LogData, "Test.Logs", "Description", GlueTableHourly, partitionTestEvent{})

	// test partition exists at start
	glueClient := &mockGlue{}
	glueClient.On("GetTable", mock.Anything).Return(testGetTableOutput, nil)
	glueClient.On("CreatePartition", mock.Anything).Return(testCreatePartitionOutput, entityExistsError)
	err := gm.CreateJSONPartition(glueClient, refTime)
	assert.NoError(t, err)
	glueClient.AssertExpectations(t)
}

func TestCreateJSONPartitionErrorGettingTable(t *testing.T) {
	gm := NewGlueTableMetadata(models.LogData, "Test.Logs", "Description", GlueTableHourly, partitionTestEvent{})
	// test error in GetTable
	glueClient := &mockGlue{}
	glueClient.On("GetTable", mock.Anything).Return(testGetTableOutput, nonAWSError).Once()
	err := gm.CreateJSONPartition(glueClient, refTime)
	assert.Error(t, err)
	assert.Equal(t, nonAWSError, err)
	glueClient.AssertExpectations(t)
}

func TestCreateJSONPartitionNon(t *testing.T) {
	gm := NewGlueTableMetadata(models.LogData, "Test.Logs", "Description", GlueTableHourly, partitionTestEvent{})
	// test error in CreatePartition
	glueClient := &mockGlue{}
	glueClient.On("GetTable", mock.Anything).Return(testGetTableOutput, nil).Once()
	glueClient.On("CreatePartition", mock.Anything).Return(testCreatePartitionOutput, nonAWSError).Once()
	err := gm.CreateJSONPartition(glueClient, refTime)
	assert.Error(t, err)
	assert.Equal(t, nonAWSError, err)
	glueClient.AssertExpectations(t)
}

func TestSyncPartitionPartitionDoesntExist(t *testing.T) {
	gm := NewGlueTableMetadata(models.LogData, "Test.Logs", "Description", GlueTableHourly, partitionTestEvent{})

	// test not exists error in DeletePartition (should not fail)
	glueClient := &mockGlue{}
	glueClient.On("DeletePartition", mock.Anything).Return(testDeletePartitionOutput, entityNotFoundError).Once()
	glueClient.On("GetTable", mock.Anything).Return(testGetTableOutput, nil).Once()
	glueClient.On("CreatePartition", mock.Anything).Return(testCreatePartitionOutput, nil).Once()
	err := gm.SyncPartition(glueClient, refTime)
	assert.NoError(t, err)
	glueClient.AssertExpectations(t)
}

func TestSyncPartitionDeletePartitionOtherAWSError(t *testing.T) {
	gm := NewGlueTableMetadata(models.LogData, "Test.Logs", "Description", GlueTableHourly, partitionTestEvent{})
	// test other AWS error in DeletePartition (should fail)
	glueClient := &mockGlue{}
	glueClient.On("DeletePartition", mock.Anything).Return(testDeletePartitionOutput, otherAWSError).Once()
	err := gm.SyncPartition(glueClient, refTime)
	assert.Error(t, err)
	assert.Equal(t, otherAWSError.Error(), errors.Cause(err).Error())
	glueClient.AssertExpectations(t)
}

func TestSyncPartitionDeletePartitionNonAWSError(t *testing.T) {
	gm := NewGlueTableMetadata(models.LogData, "Test.Logs", "Description", GlueTableHourly, partitionTestEvent{})
	// test non AWS error in DeletePartition (should fail)
	glueClient := &mockGlue{}
	glueClient.On("DeletePartition", mock.Anything).Return(testDeletePartitionOutput, nonAWSError).Once()
	err := gm.SyncPartition(glueClient, refTime)
	assert.Error(t, err)
	assert.Equal(t, nonAWSError.Error(), errors.Cause(err).Error())
	glueClient.AssertExpectations(t)
}

type mockGlue struct {
	glueiface.GlueAPI
	mock.Mock
}

func (m *mockGlue) GetPartition(input *glue.GetPartitionInput) (*glue.GetPartitionOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*glue.GetPartitionOutput), args.Error(1)
}

func (m *mockGlue) GetTable(input *glue.GetTableInput) (*glue.GetTableOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*glue.GetTableOutput), args.Error(1)
}

func (m *mockGlue) CreatePartition(input *glue.CreatePartitionInput) (*glue.CreatePartitionOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*glue.CreatePartitionOutput), args.Error(1)
}

func (m *mockGlue) DeletePartition(input *glue.DeletePartitionInput) (*glue.DeletePartitionOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*glue.DeletePartitionOutput), args.Error(1)
}
