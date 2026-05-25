/*
 * smoke.c — end-to-end smoke test for libhelios.a.
 *
 * Build (from repo root, after `go build -buildmode=c-archive ...`):
 *   cc -o bindings/c/smoke bindings/c/smoke.c bindings/c/libhelios.a \
 *      -lpthread -ldl
 *
 * Verifies: new -> write -> commit -> branch -> fork -> overlay write ->
 * merge -> branch head advanced -> free. Exits 0 on success, non-zero on fail.
 */

#include <stdio.h>
#include <string.h>
#include <stdlib.h>
#include "helios.h"

#define CHECK(expr, msg)                                                   \
	do {                                                                   \
		int _rc = (expr);                                                  \
		if (_rc != HELIOS_OK) {                                            \
			fprintf(stderr, "FAIL %s: rc=%d\n", msg, _rc);                 \
			return 1;                                                      \
		}                                                                  \
	} while (0)

int main(void) {
	helios_vst_t v = helios_vst_new();
	if (v == 0) { fprintf(stderr, "FAIL vst_new returned 0\n"); return 1; }

	const char seed[] = "hello, ionq";
	CHECK(helios_vst_write_file(v, (char*)"greeting.txt",
	                            (unsigned char*)seed, strlen(seed)),
	      "vst_write_file");

	/* Read it back. */
	unsigned char *buf = NULL; size_t buflen = 0;
	CHECK(helios_vst_read_file(v, (char*)"greeting.txt", &buf, &buflen),
	      "vst_read_file");
	if (buflen != strlen(seed) || memcmp(buf, seed, buflen) != 0) {
		fprintf(stderr, "FAIL read mismatch (%zu bytes)\n", buflen);
		return 1;
	}
	helios_buffer_free(buf);

	/* Commit. */
	char *snap_id = NULL; size_t snap_len = 0;
	CHECK(helios_vst_commit(v, (char*)"seed", &snap_id, &snap_len), "vst_commit");

	/* Create a branch at that snapshot. */
	CHECK(helios_vst_create_branch(v, (char*)"main", snap_id, snap_len),
	      "vst_create_branch");

	/* Fork off the branch head. */
	helios_fork_t f = 0;
	CHECK(helios_fork_new(v, snap_id, snap_len, &f), "fork_new");
	if (f == 0) { fprintf(stderr, "FAIL fork_new returned 0\n"); return 1; }

	const char payload[] = "overlay payload";
	CHECK(helios_fork_write(f, (char*)"overlay.txt",
	                        (unsigned char*)payload, strlen(payload)),
	      "fork_write");

	/* Read overlay value through the fork. */
	unsigned char *fbuf = NULL; size_t flen = 0;
	CHECK(helios_fork_read(f, (char*)"overlay.txt", &fbuf, &flen),
	      "fork_read overlay");
	if (flen != strlen(payload) || memcmp(fbuf, payload, flen) != 0) {
		fprintf(stderr, "FAIL fork read overlay mismatch\n");
		return 1;
	}
	helios_buffer_free(fbuf);

	/* Merge into branch. */
	char *new_id = NULL; size_t new_len = 0;
	CHECK(helios_fork_merge_into(f, (char*)"main", &new_id, &new_len),
	      "fork_merge_into");

	/* Verify head advanced. */
	char *head = NULL; size_t head_len = 0;
	CHECK(helios_vst_branch_head(v, (char*)"main", &head, &head_len),
	      "vst_branch_head");
	if (head_len != new_len || memcmp(head, new_id, head_len) != 0) {
		fprintf(stderr, "FAIL branch head != merged id\n");
		return 1;
	}
	if (head_len == snap_len && memcmp(head, snap_id, head_len) == 0) {
		fprintf(stderr, "FAIL branch head did not advance\n");
		return 1;
	}

	helios_string_free(head);
	helios_string_free(new_id);
	helios_string_free(snap_id);

	CHECK(helios_vst_free(v), "vst_free");
	printf("OK\n");
	return 0;
}
