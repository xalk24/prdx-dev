import type { ContractDeck } from './contract';

export type JobStatus = 'queued' | 'running' | 'succeeded' | 'failed' | 'cancelled';
export interface Job {
	id: string;
	projectId: string;
	type: 'generation' | 'export';
	status: JobStatus;
	baseRevision?: number;
	error?: { code: string; message: string; retryable: boolean };
}
export interface DeckSnapshot {
	deck: ContractDeck;
	revision: number;
}
export interface Project {
	id: string;
	title: string;
	revision: number;
}

export function projectAssetUrl(projectId: string, assetId: string, base = '/api/v1') {
	return `${base}/projects/${encodeURIComponent(projectId)}/assets/${encodeURIComponent(assetId)}`;
}

export class ApiError extends Error {
	constructor(
		public status: number,
		public code: string,
		message: string,
		public retryable = false
	) {
		super(message);
	}
}

export class PresentatorApi {
	constructor(
		private request: typeof fetch = fetch,
		private base = '/api/v1'
	) {}
	private async json<T>(url: string, init?: RequestInit): Promise<T> {
		const response = await this.request(`${this.base}${url}`, init);
		if (!response.ok) {
			const body = await response.json().catch(() => ({
				error: { code: 'unknown', message: response.statusText, retryable: false }
			}));
			throw new ApiError(
				response.status,
				body.error?.code ?? 'unknown',
				body.error?.message ?? response.statusText,
				body.error?.retryable ?? false
			);
		}
		return response.json() as Promise<T>;
	}
	getDeck(projectId: string) {
		return this.json<DeckSnapshot>(`/projects/${projectId}/deck`);
	}
	createProject(title: string) {
		return this.json<Project>('/projects', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ title })
		});
	}
	saveDeck(projectId: string, deck: ContractDeck, revision: number) {
		return this.json<DeckSnapshot>(`/projects/${projectId}/deck`, {
			method: 'PUT',
			headers: { 'Content-Type': 'application/json', 'If-Match': String(revision) },
			body: JSON.stringify(deck)
		});
	}
	createGeneration(projectId: string, brief: string, key: string = crypto.randomUUID()) {
		return this.json<Job>(`/projects/${projectId}/generation-jobs`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json', 'Idempotency-Key': key },
			body: JSON.stringify({ brief, locale: 'ru', profile: 'default' })
		});
	}
	getJob(jobId: string) {
		return this.json<Job>(`/generation-jobs/${jobId}`);
	}
	getExportJob(jobId: string) {
		return this.json<Job>(`/export-jobs/${jobId}`);
	}
	getCandidate(jobId: string) {
		return this.json<ContractDeck>(`/generation-jobs/${jobId}/candidate`);
	}
	applyCandidate(jobId: string, baseRevision: number, key: string = crypto.randomUUID()) {
		return this.json<DeckSnapshot>(`/generation-jobs/${jobId}/apply`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json', 'Idempotency-Key': key },
			body: JSON.stringify({ baseRevision })
		});
	}
	createExport(projectId: string, revision: number, key: string = crypto.randomUUID()) {
		return this.json<Job>(`/projects/${projectId}/export-jobs`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json', 'Idempotency-Key': key },
			body: JSON.stringify({ revision })
		});
	}
	async downloadExport(jobId: string) {
		const response = await this.request(`${this.base}/export-jobs/${jobId}/download`);
		if (!response.ok)
			throw new ApiError(
				response.status,
				'export_download_failed',
				'Could not download PDF.',
				true
			);
		return response.blob();
	}
}

export async function waitForJob(
	api: PresentatorApi,
	id: string,
	wait = (ms: number) => new Promise((r) => setTimeout(r, ms)),
	type: 'generation' | 'export' = 'generation'
): Promise<Job> {
	for (let attempt = 0; attempt < 60; attempt++) {
		const job = type === 'export' ? await api.getExportJob(id) : await api.getJob(id);
		if (!['queued', 'running'].includes(job.status)) return job;
		await wait(1000);
	}
	throw new ApiError(408, 'job_timeout', 'The job is still running.', true);
}
