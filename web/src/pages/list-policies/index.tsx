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
import { Alert, Box, Card } from 'pouncejs';
import { DEFAULT_LARGE_PAGE_SIZE } from 'Source/constants';
import { useQuery, gql } from '@apollo/client';
import { convertObjArrayValuesToCsv, extractErrorMessage } from 'Helpers/utils';
import {
  ListPoliciesInput,
  ListPoliciesResponse,
  SortDirEnum,
  ListPoliciesSortFieldsEnum,
} from 'Generated/schema';
import TablePaginationControls from 'Components/utils/table-pagination-controls';
import useRequestParamsWithPagination from 'Hooks/useRequestParamsWithPagination';
import ErrorBoundary from 'Components/error-boundary';
import isEmpty from 'lodash-es/isEmpty';
import ListPoliciesTable from './subcomponents/list-policies-table';
import ListPoliciesActions from './subcomponents/list-policies-actions';
import ListPoliciesPageSkeleton from './skeleton';
import ListPoliciesPageEmptyDataFallback from './empty-data-fallback';

export const LIST_POLICIES = gql`
  query ListPolicies($input: ListPoliciesInput) {
    policies(input: $input) {
      policies {
        complianceStatus
        lastModified
        resourceTypes
        severity
        id
        displayName
        enabled
      }
      paging {
        totalPages
        thisPage
        totalItems
      }
    }
  }
`;

interface ApolloData {
  policies: ListPoliciesResponse;
}
interface ApolloVariables {
  input: ListPoliciesInput;
}

const ListPolicies = () => {
  const {
    requestParams,
    updateRequestParamsAndResetPaging,
    updatePagingParams,
  } = useRequestParamsWithPagination<ListPoliciesInput>();

  const { loading, error, data } = useQuery<ApolloData, ApolloVariables>(LIST_POLICIES, {
    fetchPolicy: 'cache-and-network',
    variables: {
      input: convertObjArrayValuesToCsv(requestParams),
    },
  });

  if (loading && !data) {
    return <ListPoliciesPageSkeleton />;
  }

  if (error) {
    return (
      <Alert
        mb={6}
        variant="error"
        title="Couldn't load your policies"
        description={
          extractErrorMessage(error) ||
          'There was an error when performing your request, please contact support@runpanther.io'
        }
      />
    );
  }

  const policyItems = data.policies.policies;
  const pagingData = data.policies.paging;

  if (!policyItems.length && isEmpty(requestParams)) {
    return <ListPoliciesPageEmptyDataFallback />;
  }

  return (
    <React.Fragment>
      <ListPoliciesActions />
      <ErrorBoundary>
        <Card>
          <ListPoliciesTable
            enumerationStartIndex={(pagingData.thisPage - 1) * DEFAULT_LARGE_PAGE_SIZE}
            items={policyItems}
            onSort={updateRequestParamsAndResetPaging}
            sortBy={requestParams.sortBy || ListPoliciesSortFieldsEnum.Id}
            sortDir={requestParams.sortDir || SortDirEnum.Ascending}
          />
        </Card>
      </ErrorBoundary>
      <Box my={6}>
        <TablePaginationControls
          page={pagingData.thisPage}
          totalPages={pagingData.totalPages}
          onPageChange={updatePagingParams}
        />
      </Box>
    </React.Fragment>
  );
};

export default ListPolicies;
