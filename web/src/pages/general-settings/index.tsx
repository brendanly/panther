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

import React from 'react';
import { Alert, Box, useSnackbar, Text, Flex } from 'pouncejs';
import { useQuery, gql, useMutation } from '@apollo/client';
import { GeneralSettings, UpdateGeneralSettingsInput } from 'Generated/schema';
import ErrorBoundary from 'Components/error-boundary';
import { extractErrorMessage } from 'Helpers/utils';
import CompanyInformationForm from 'Components/forms/company-information-form';
import Panel from 'Components/panel';
import GeneralSettingsPageSkeleton from './skeleton';

export const GET_GENERAL_SETTINGS = gql`
  query GetGeneralSettings {
    generalSettings {
      displayName
      email
      errorReportingConsent
    }
  }
`;

const UPDATE_GENERAL_SETTINGS = gql`
  mutation UpdateGeneralSettings($input: UpdateGeneralSettingsInput!) {
    updateGeneralSettings(input: $input) {
      displayName
      email
      errorReportingConsent
    }
  }
`;

interface ApolloMutationInput {
  input: UpdateGeneralSettingsInput;
}

interface ApolloMutationData {
  updateGeneralSettings: Pick<GeneralSettings, 'displayName' | 'email' | 'errorReportingConsent'>;
}

interface ApolloQueryData {
  generalSettings: GeneralSettings;
}

// Parent container for the general settings section
const GeneralSettingsContainer: React.FC = () => {
  const { pushSnackbar } = useSnackbar();

  const {
    loading: getGeneralSettingsLoading,
    error: getGeneralSettingsError,
    data: getGeneralSettingsData,
  } = useQuery<ApolloQueryData>(GET_GENERAL_SETTINGS);

  const [
    updateGeneralSettings,
    { error: updateGeneralSettingsError, data: updateGeneralSettingsData },
  ] = useMutation<ApolloMutationData, ApolloMutationInput>(UPDATE_GENERAL_SETTINGS);

  React.useEffect(() => {
    if (updateGeneralSettingsData) {
      pushSnackbar({ variant: 'success', title: `Successfully updated company information` });
    }
  }, [updateGeneralSettingsData]);

  React.useEffect(() => {
    if (updateGeneralSettingsError) {
      pushSnackbar({
        variant: 'error',
        title:
          extractErrorMessage(updateGeneralSettingsError) ||
          'Failed to update company information due to an unknown error',
      });
    }
  }, [updateGeneralSettingsError]);

  if (getGeneralSettingsLoading) {
    return <GeneralSettingsPageSkeleton />;
  }

  if (getGeneralSettingsError) {
    return (
      <Alert
        variant="error"
        title="Failed to query company information"
        description={
          extractErrorMessage(getGeneralSettingsError) ||
          'Sorry, something went wrong, please reach out to support@runpanther.io if this problem persists'
        }
      />
    );
  }

  const { displayName, email, errorReportingConsent } = getGeneralSettingsData.generalSettings;
  return (
    <Box mb={6}>
      <ErrorBoundary>
        <Box mb={2}>
          <Panel title="About Panther" size="large">
            <Box width={500} m="auto">
              <Flex mb={6}>
                <Text color="grey300" size="large" width={80}>
                  Plan
                </Text>
                <Text color="grey500" size="large" fontWeight="bold">
                  Community
                </Text>
              </Flex>
              <Flex>
                <Text color="grey300" size="large" width={80}>
                  Version
                </Text>
                <Text color="grey500" size="large" fontWeight="bold">
                  {process.env.PANTHER_VERSION || 'N/A'}
                </Text>
              </Flex>
            </Box>
          </Panel>
        </Box>
        <Panel title="General Settings" size="large">
          <Box width={500} mx="auto" mt={10}>
            <CompanyInformationForm
              initialValues={{
                displayName,
                email,
                errorReportingConsent,
              }}
              onSubmit={values => updateGeneralSettings({ variables: { input: values } })}
            />
          </Box>
        </Panel>
      </ErrorBoundary>
    </Box>
  );
};

export default GeneralSettingsContainer;
