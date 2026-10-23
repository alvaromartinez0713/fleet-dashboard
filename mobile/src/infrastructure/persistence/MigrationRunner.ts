import {
  POSITIONS_TABLE_SCHEMA,
  POSITIONS_INDEXES,
  MIGRATION_VERSION,
} from './SQLiteSchema';

/**
 * MigrationRunner
 *
 * Handles database schema initialization and migrations on app startup.
 * Ensures the positions table exists with all required columns and indexes.
 */
export class MigrationRunner {
  /**
   * Run all pending migrations
   *
   * @param executeSql Function to execute raw SQL (e.g., db.exec from sqlite3)
   */
  async runMigrations(executeSql: (sql: string) => Promise<void>): Promise<void> {
    try {
      // 1. Create positions table if it doesn't exist
      await executeSql(POSITIONS_TABLE_SCHEMA);

      // 2. Create all indexes
      for (const indexSql of POSITIONS_INDEXES) {
        await executeSql(indexSql);
      }

      console.log(`[MigrationRunner] Migrations completed (version ${MIGRATION_VERSION})`);
    } catch (error) {
      console.error('[MigrationRunner] Migration failed:', error);
      throw new Error(`Database migration failed: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  }

  /**
   * Validate that the schema is properly initialized
   *
   * @param querySql Function to execute a SELECT query and return rows
   * @returns true if schema is valid, false otherwise
   */
  async validateSchema(querySql: (sql: string) => Promise<any[]>): Promise<boolean> {
    try {
      const result = await querySql(`
        SELECT name FROM sqlite_master 
        WHERE type='table' AND name='positions';
      `);
      return result && result.length > 0;
    } catch (error) {
      console.error('[MigrationRunner] Schema validation failed:', error);
      return false;
    }
  }

  /**
   * Get database info for debugging
   */
  async getDatabaseInfo(querySql: (sql: string) => Promise<any[]>): Promise<{
    tableExists: boolean;
    recordCount: number;
    statusCounts: Record<string, number>;
  }> {
    try {
      const tableCheck = await querySql(`
        SELECT name FROM sqlite_master 
        WHERE type='table' AND name='positions';
      `);

      if (!tableCheck || tableCheck.length === 0) {
        return {
          tableExists: false,
          recordCount: 0,
          statusCounts: {},
        };
      }

      const countResult = await querySql('SELECT COUNT(*) as count FROM positions;');
      const recordCount = countResult[0]?.count || 0;

      const statusResult = await querySql(
        'SELECT status, COUNT(*) as count FROM positions GROUP BY status;'
      );

      const statusCounts: Record<string, number> = {};
      if (statusResult) {
        for (const row of statusResult) {
          statusCounts[row.status] = row.count;
        }
      }

      return {
        tableExists: true,
        recordCount,
        statusCounts,
      };
    } catch (error) {
      console.error('[MigrationRunner] Failed to get database info:', error);
      return {
        tableExists: false,
        recordCount: 0,
        statusCounts: {},
      };
    }
  }
}
