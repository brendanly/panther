package processor

/**
 * Panther is a scalable, powerful, cloud-native SIEM written in Golang/React.
 * Copyright (C) 2020 Panther Labs Inc
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/lambda/lambdaiface"
	"github.com/aws/aws-sdk-go/service/s3"
	"go.uber.org/zap"

	"github.com/panther-labs/panther/api/lambda/snapshot/models"
	"github.com/panther-labs/panther/pkg/genericapi"
)

const (
	cweAccountTimeout       = 15 * time.Minute
	refreshInterval         = 2 * time.Minute
	snapshotAPIFunctionName = "panther-snapshot-api"
)

var (
	// Valid accounts, keyed on accountID
	accounts            = make(map[string]*models.SourceIntegration)
	accountsLastUpdated time.Time

	// Accounts where CloudWatch Events are enabled, and should be preferred over S3 notifications
	// Keyed by accountID + region
	cweAccounts = make(map[string]time.Time)

	// Setup the clients to talk to the Snapshot API
	sess                               = session.Must(session.NewSession())
	lambdaClient lambdaiface.LambdaAPI = lambda.New(sess)
	s3Svc                              = s3.New(sess)
)

func resetAccountCache() {
	accounts = make(map[string]*models.SourceIntegration)
}

// checkCWECache looks up an accountId in the cweAccounts cache and returns true only if the accountId is present and
// not expired. This is to prevent a long delay in clearing the cache after removing a CWE configuration.
func checkCWECache(key string) bool {
	if timestamp, ok := cweAccounts[key]; ok {
		if time.Since(timestamp) > cweAccountTimeout {
			delete(cweAccounts, key)
			return false
		}
		return true
	}
	return false
}

func refreshAccounts() error {
	if len(accounts) != 0 && accountsLastUpdated.Add(refreshInterval).After(time.Now()) {
		zap.L().Debug("using cached accounts")
		return nil
	}

	zap.L().Debug("populating account cache")
	input := &models.LambdaInput{
		ListIntegrations: &models.ListIntegrationsInput{
			IntegrationType: aws.String("aws-scan"),
		},
	}
	var output []*models.SourceIntegration
	err := genericapi.Invoke(lambdaClient, snapshotAPIFunctionName, input, &output)
	if err != nil {
		return err
	}

	for _, integration := range output {
		accounts[*integration.AWSAccountID] = integration
	}
	accountsLastUpdated = time.Now()

	return nil
}
