import { describe, expect, it, vi } from 'vitest';
import { PresentatorApi, waitForJob } from './api';
import type { ContractDeck } from './contract';

const deck = {
	schemaVersion: '1.0',
	deckId: 'deck-demo',
	canvas: { width: 1920, height: 1080 },
	theme: { background: '#FFFFFF' },
	slides: [{ id: 'slide-1', name: 'Opening', elements: [] }],
	comments: []
} as ContractDeck;
const response = (body: unknown, status = 200) =>
	new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } });

describe('Presentator API contract', () => {
	it('saves against the acknowledged revision using If-Match', async () => {
		const fetcher = vi.fn(async () => response({ deck, revision: 13 }));
		const api = new PresentatorApi(fetcher as typeof fetch);
		await expect(api.saveDeck('project-1', deck, 12)).resolves.toMatchObject({ revision: 13 });
		expect(fetcher).toHaveBeenCalledWith(
			'/api/v1/projects/project-1/deck',
			expect.objectContaining({
				method: 'PUT',
				headers: expect.objectContaining({ 'If-Match': '12' })
			})
		);
	});
	it('keeps candidate and apply as separate idempotent operations', async () => {
		const fetcher = vi
			.fn()
			.mockResolvedValueOnce(response(deck))
			.mockResolvedValueOnce(response({ deck, revision: 13 }));
		const api = new PresentatorApi(fetcher as typeof fetch);
		await api.getCandidate('job-1');
		await api.applyCandidate('job-1', 12, 'apply-once');
		expect(fetcher.mock.calls[0][0]).toBe('/api/v1/generation-jobs/job-1/candidate');
		expect(fetcher.mock.calls[1][1]).toEqual(
			expect.objectContaining({
				method: 'POST',
				headers: expect.objectContaining({ 'Idempotency-Key': 'apply-once' }),
				body: JSON.stringify({ baseRevision: 12 })
			})
		);
	});
	it('creates export for an immutable saved revision', async () => {
		const fetcher = vi.fn(async () =>
			response({ id: 'export-1', projectId: 'project-1', type: 'export', status: 'queued' })
		);
		const api = new PresentatorApi(fetcher as typeof fetch);
		await api.createExport('project-1', 13, 'export-once');
		expect(fetcher).toHaveBeenCalledWith(
			'/api/v1/projects/project-1/export-jobs',
			expect.objectContaining({ body: JSON.stringify({ revision: 13 }) })
		);
	});
	it('polls queued/running jobs and exposes stable failure envelopes', async () => {
		const fetcher = vi
			.fn()
			.mockResolvedValueOnce(
				response({ id: 'j', projectId: 'p', type: 'generation', status: 'running' })
			)
			.mockResolvedValueOnce(
				response({
					id: 'j',
					projectId: 'p',
					type: 'generation',
					status: 'failed',
					error: { code: 'predictor_unavailable', message: 'Unavailable', retryable: true }
				})
			);
		const api = new PresentatorApi(fetcher as typeof fetch);
		const job = await waitForJob(api, 'j', async () => {});
		expect(job.status).toBe('failed');
		const failing = new PresentatorApi(
			async () =>
				response(
					{
						error: { code: 'revision_conflict', message: 'Reload', retryable: true },
						correlationId: 'c'
					},
					409
				) as never
		);
		await expect(failing.getDeck('p')).rejects.toEqual(
			expect.objectContaining({ status: 409, code: 'revision_conflict', retryable: true })
		);
	});
});
