#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include "lbug.h"

// --- High Level LadybugDB Functions ---

// ldbug_open creates or opens an embedded LadybugDB (.lddb) instance and connection.
int ldbug_open(const char* path, void** out_db, void** out_conn) {
    lbug_system_config cfg = lbug_default_system_config();
    // CHANGE HERE: Buffer pool size in bytes (default: 64MB). Increase for higher caching/large repos.
    cfg.buffer_pool_size = 64 * 1024 * 1024;
    // CHANGE HERE: Max worker threads (0 = auto-detect based on available CPU cores).
    cfg.max_num_threads = 0;
    // CHANGE HERE: Enable or disable on-disk page compression.
    cfg.enable_compression = true;

    lbug_database* db = (lbug_database*)malloc(sizeof(lbug_database));
    if (lbug_database_init(path, cfg, db) != LbugSuccess) {
        free(db);
        return -1;
    }

    lbug_connection* conn = (lbug_connection*)malloc(sizeof(lbug_connection));
    if (lbug_connection_init(db, conn) != LbugSuccess) {
        lbug_database_destroy(db);
        free(db);
        free(conn);
        return -1;
    }

    // Initialize key-value node table if it doesn't already exist
    lbug_query_result res;
    lbug_connection_query(conn, "CREATE NODE TABLE IF NOT EXISTS KV(key STRING, val STRING, PRIMARY KEY(key));", &res);
    lbug_query_result_destroy(&res);

    *out_db = db;
    *out_conn = conn;
    return 0;
}

// ldbug_close cleans up the database connection and database instance.
void ldbug_close(void* db, void* conn) {
    if (conn) {
        lbug_connection_destroy((lbug_connection*)conn);
        free(conn);
    }
    if (db) {
        lbug_database_destroy((lbug_database*)db);
        free(db);
    }
}

// ldbug_get retrieves the value string associated with the key.
// Returns 0 on success (found), 1 if not found, -1 on error.
int ldbug_get(void* conn, const char* key, char** out_val) {
    lbug_prepared_statement prep;
    if (lbug_connection_prepare((lbug_connection*)conn, "MATCH (n:KV {key: $k}) RETURN n.val", &prep) != LbugSuccess) {
        return -1;
    }
    lbug_prepared_statement_bind_string(&prep, "k", key);
    lbug_query_result res;
    lbug_connection_execute((lbug_connection*)conn, &prep, &res);

    int status = 1; // 1 = key not found
    *out_val = NULL;

    if (lbug_query_result_is_success(&res) && lbug_query_result_has_next(&res)) {
        lbug_flat_tuple tup;
        if (lbug_query_result_get_next(&res, &tup) == LbugSuccess) {
            lbug_value val;
            if (lbug_flat_tuple_get_value(&tup, 0, &val) == LbugSuccess) {
                char* s = NULL;
                if (lbug_value_get_string(&val, &s) == LbugSuccess && s) {
                    *out_val = strdup(s);
                    lbug_destroy_string(s);
                    status = 0;
                }
                lbug_value_destroy(&val);
            }
            lbug_flat_tuple_destroy(&tup);
        }
    }

    lbug_query_result_destroy(&res);
    lbug_prepared_statement_destroy(&prep);
    return status;
}

// ldbug_set upserts a key-value record into the KV table.
int ldbug_set(void* conn, const char* key, const char* val) {
    lbug_prepared_statement prep;
    if (lbug_connection_prepare((lbug_connection*)conn,
        "MERGE (n:KV {key: $k}) ON CREATE SET n.val = $v ON MATCH SET n.val = $v", &prep) != LbugSuccess) {
        return -1;
    }
    lbug_prepared_statement_bind_string(&prep, "k", key);
    lbug_prepared_statement_bind_string(&prep, "v", val);
    lbug_query_result res;
    lbug_connection_execute((lbug_connection*)conn, &prep, &res);
    int ok = lbug_query_result_is_success(&res) ? 0 : -1;
    lbug_query_result_destroy(&res);
    lbug_prepared_statement_destroy(&prep);
    return ok;
}

// ldbug_delete deletes a key-value record by key.
int ldbug_delete(void* conn, const char* key) {
    lbug_prepared_statement prep;
    if (lbug_connection_prepare((lbug_connection*)conn, "MATCH (n:KV {key: $k}) DELETE n", &prep) != LbugSuccess) {
        return -1;
    }
    lbug_prepared_statement_bind_string(&prep, "k", key);
    lbug_query_result res;
    lbug_connection_execute((lbug_connection*)conn, &prep, &res);
    int ok = lbug_query_result_is_success(&res) ? 0 : -1;
    lbug_query_result_destroy(&res);
    lbug_prepared_statement_destroy(&prep);
    return ok;
}

// ldbug_begin starts a transaction for atomic batch operations.
int ldbug_begin(void* conn) {
    lbug_query_result res;
    lbug_connection_query((lbug_connection*)conn, "BEGIN TRANSACTION", &res);
    int ok = lbug_query_result_is_success(&res) ? 0 : -1;
    lbug_query_result_destroy(&res);
    return ok;
}

// ldbug_commit commits the active transaction.
int ldbug_commit(void* conn) {
    lbug_query_result res;
    lbug_connection_query((lbug_connection*)conn, "COMMIT", &res);
    int ok = lbug_query_result_is_success(&res) ? 0 : -1;
    lbug_query_result_destroy(&res);
    return ok;
}

// ldbug_rollback rolls back the active transaction.
int ldbug_rollback(void* conn) {
    lbug_query_result res;
    lbug_connection_query((lbug_connection*)conn, "ROLLBACK", &res);
    int ok = lbug_query_result_is_success(&res) ? 0 : -1;
    lbug_query_result_destroy(&res);
    return ok;
}

// ldbug_count_prefix counts records where key begins with the specified prefix.
int ldbug_count_prefix(void* conn, const char* prefix, int* out_count) {
    lbug_prepared_statement prep;
    if (lbug_connection_prepare((lbug_connection*)conn,
        "MATCH (n:KV) WHERE starts_with(n.key, $p) RETURN count(n)", &prep) != LbugSuccess) {
        return -1;
    }
    lbug_prepared_statement_bind_string(&prep, "p", prefix);
    lbug_query_result res;
    lbug_connection_execute((lbug_connection*)conn, &prep, &res);

    int ok = -1;
    if (lbug_query_result_is_success(&res) && lbug_query_result_has_next(&res)) {
        lbug_flat_tuple tup;
        if (lbug_query_result_get_next(&res, &tup) == LbugSuccess) {
            lbug_value val;
            if (lbug_flat_tuple_get_value(&tup, 0, &val) == LbugSuccess) {
                int64_t cnt = 0;
                lbug_value_get_int64(&val, &cnt);
                *out_count = (int)cnt;
                ok = 0;
                lbug_value_destroy(&val);
            }
            lbug_flat_tuple_destroy(&tup);
        }
    }
    lbug_query_result_destroy(&res);
    lbug_prepared_statement_destroy(&prep);
    return ok;
}

// ldbug_scan_prefix retrieves keys and values matching the prefix up to the limit.
int ldbug_scan_prefix(void* conn, const char* prefix, int limit, char*** out_keys, char*** out_vals, int* out_count) {
    // CHANGE HERE: Default maximum results to scan when limit is <= 0.
    if (limit <= 0) limit = 10000;

    lbug_prepared_statement prep;
    if (lbug_connection_prepare((lbug_connection*)conn,
        "MATCH (n:KV) WHERE starts_with(n.key, $p) RETURN n.key, n.val LIMIT $lim", &prep) != LbugSuccess) {
        return -1;
    }
    lbug_prepared_statement_bind_string(&prep, "p", prefix);
    lbug_prepared_statement_bind_int64(&prep, "lim", (int64_t)limit);
    lbug_query_result res;
    lbug_connection_execute((lbug_connection*)conn, &prep, &res);

    int count = 0;
    *out_keys = NULL;
    *out_vals = NULL;
    *out_count = 0;

    if (lbug_query_result_is_success(&res)) {
        uint64_t num_tuples = lbug_query_result_get_num_tuples(&res);
        if (num_tuples > 0) {
            *out_keys = (char**)malloc(sizeof(char*) * num_tuples);
            *out_vals = (char**)malloc(sizeof(char*) * num_tuples);
            while (lbug_query_result_has_next(&res)) {
                lbug_flat_tuple tup;
                if (lbug_query_result_get_next(&res, &tup) == LbugSuccess) {
                    lbug_value k_val, v_val;
                    char *k_s = NULL, *v_s = NULL;
                    if (lbug_flat_tuple_get_value(&tup, 0, &k_val) == LbugSuccess) {
                        lbug_value_get_string(&k_val, &k_s);
                        lbug_value_destroy(&k_val);
                    }
                    if (lbug_flat_tuple_get_value(&tup, 1, &v_val) == LbugSuccess) {
                        lbug_value_get_string(&v_val, &v_s);
                        lbug_value_destroy(&v_val);
                    }
                    (*out_keys)[count] = k_s ? strdup(k_s) : strdup("");
                    (*out_vals)[count] = v_s ? strdup(v_s) : strdup("");
                    if (k_s) lbug_destroy_string(k_s);
                    if (v_s) lbug_destroy_string(v_s);
                    count++;
                    lbug_flat_tuple_destroy(&tup);
                }
            }
        }
        *out_count = count;
    }
    lbug_query_result_destroy(&res);
    lbug_prepared_statement_destroy(&prep);
    return 0;
}

// ldbug_query_cypher executes arbitrary Cypher queries and flattens result rows into a 1D cell array.
int ldbug_query_cypher(void* conn, const char* cypher, char*** out_cells, int* out_rows, int* out_cols, char** out_err) {
    lbug_query_result res;
    lbug_connection_query((lbug_connection*)conn, cypher, &res);
    *out_err = NULL;

    if (!lbug_query_result_is_success(&res)) {
        char* err = lbug_query_result_get_error_message(&res);
        if (err) {
            *out_err = strdup(err);
            lbug_destroy_string(err);
        } else {
            *out_err = strdup("unknown error");
        }
        lbug_query_result_destroy(&res);
        return -1;
    }

    int cols = (int)lbug_query_result_get_num_columns(&res);
    int rows = (int)lbug_query_result_get_num_tuples(&res);
    *out_cols = cols;
    *out_rows = rows;
    *out_cells = NULL;

    if (rows > 0 && cols > 0) {
        *out_cells = (char**)malloc(sizeof(char*) * rows * cols);
        int r = 0;
        while (lbug_query_result_has_next(&res) && r < rows) {
            lbug_flat_tuple tup;
            if (lbug_query_result_get_next(&res, &tup) == LbugSuccess) {
                for (int c = 0; c < cols; c++) {
                    lbug_value v;
                    if (lbug_flat_tuple_get_value(&tup, c, &v) == LbugSuccess) {
                        char* s = lbug_value_to_string(&v);
                        (*out_cells)[r * cols + c] = s ? strdup(s) : strdup("");
                        if (s) lbug_destroy_string(s);
                        lbug_value_destroy(&v);
                    } else {
                        (*out_cells)[r * cols + c] = strdup("");
                    }
                }
                lbug_flat_tuple_destroy(&tup);
                r++;
            }
        }
    }
    lbug_query_result_destroy(&res);
    return 0;
}

// ldbug_free frees single pointer memory.
void ldbug_free(void* ptr) {
    if (ptr) free(ptr);
}

// ldbug_free_string_array frees an array of strings.
void ldbug_free_string_array(char** arr, int count) {
    if (!arr) return;
    for (int i = 0; i < count; i++) {
        if (arr[i]) free(arr[i]);
    }
    free(arr);
}
