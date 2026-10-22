/**
 * MSW Server Configuration for Node.js Test Environment
 */

import { setupServer } from 'msw/node';
import { handlers } from './handlers';

export const server = setupServer(...handlers);
