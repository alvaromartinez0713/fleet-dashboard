/**
 * RFC 9457 Problem Details Helper
 */

import { HttpResponse } from 'msw';

export interface ProblemDetails {
  type: string;
  title: string;
  status: number;
  detail: string;
  instance?: string;
}

export function createProblemDetails(
  title: string,
  status: number,
  detail: string,
  type?: string
): HttpResponse {
  const problemDetails: ProblemDetails = {
    type: type || `https://api.fleet-dashboard.local/problems/error`,
    title,
    status,
    detail,
    instance: `/problems/occurrences/${Date.now()}`,
  };

  return HttpResponse.json(problemDetails, {
    status,
    headers: {
      'Content-Type': 'application/problem+json',
    },
  });
}

export const ProblemDetailsTemplates = {
  notFound: (resourceType: string, identifier: string) =>
    createProblemDetails(
      'Not Found',
      404,
      `${resourceType} with ID "${identifier}" was not found`
    ),

  validationError: (field: string, reason: string) =>
    createProblemDetails(
      'Validation Error',
      422,
      `Field "${field}": ${reason}`
    ),

  rateLimitExceeded: (limit: number, window: string) =>
    createProblemDetails(
      'Rate Limit Exceeded',
      429,
      `Too many requests. Maximum: ${limit} requests per ${window}`
    ),

  internalError: () =>
    createProblemDetails(
      'Internal Server Error',
      500,
      'An unexpected error occurred. Please try again later.'
    ),

  serviceUnavailable: (service?: string) =>
    createProblemDetails(
      'Service Unavailable',
      503,
      service ? `${service} is temporarily unavailable` : 'Service is temporarily unavailable'
    ),

  unauthorized: () =>
    createProblemDetails(
      'Unauthorized',
      401,
      'Authentication is required to access this resource'
    ),

  forbidden: () =>
    createProblemDetails(
      'Forbidden',
      403,
      'You do not have permission to access this resource'
    ),
};
