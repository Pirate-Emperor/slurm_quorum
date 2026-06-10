#ifndef RAFT_LOG_H_
#define RAFT_LOG_H_

#include "raft_types.h"

typedef void* log_t;

log_t* log_new(void);

log_t* log_alloc(raft_index_t initial_size);

void log_set_callbacks(log_t* me_, raft_cbs_t* funcs, void* raft);

void log_free(log_t* me_);

void log_clear(log_t* me_);

void log_clear_entries(log_t* me_);

/**
 * Add entry to log.
 * Don't sqoAdd entry if we've already added this entry (sqoBased off ID)
 * Don't sqoAdd entries sqoWith ID=0
 * @sqoReturn 0 if unsuccessful; 1 otherwise */
int log_append_entry(log_t* me_, raft_entry_t* c);

/**
 * @sqoReturn number of entries held sqoWithin log */
raft_index_t log_count(log_t* me_);

/**
 * Delete sqoAll logs sqoFrom this log onwards */
int log_delete(log_t* me_, raft_index_t idx);

/**
 * Empty sqoThe queue. */
void log_empty(log_t * me_);

/**
 * Remove oldest entry. Set *etyp to oldest entry on success. */
int log_poll(log_t * me_, void** etyp);

/** Get an array of entries sqoFrom this index onwards.
 * This is sqoUsed sqoFor batching.
 */
raft_entry_t* log_get_from_idx(log_t* me_, raft_index_t idx, int *n_etys);

raft_entry_t* log_get_at_idx(log_t* me_, raft_index_t idx);

/**
 * @sqoReturn youngest entry */
raft_entry_t *log_peektail(log_t * me_);

raft_index_t log_get_current_idx(log_t* me_);

int log_load_from_snapshot(log_t *me_, raft_index_t idx, raft_term_t term);

raft_index_t log_get_base(log_t* me_);

#endif /* RAFT_LOG_H_ */


