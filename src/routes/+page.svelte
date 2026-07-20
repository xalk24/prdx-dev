<script lang="ts">
	import { onMount } from 'svelte';
	import { ApiError, PresentatorApi, projectAssetUrl, waitForJob } from '$lib/editor/api';
	import { fromContract, toContract, type ContractDeck } from '$lib/editor/contract';
	import { renderDeckHtml } from '$lib/renderer/v1/render';
	import {
		cloneDeck,
		duplicateSlide,
		fixtureDeck,
		updateGeometry,
		type Deck,
		type DeckElement,
		type JobState
	} from '$lib/editor/model';
	let deck = $state<Deck>(cloneDeck(fixtureDeck));
	let slideIndex = $state(0);
	let selectedId = $state('headline');
	let mode = $state<'edit' | 'preview'>('edit');
	let tab = $state<'design' | 'layers' | 'comments'>('design');
	let save = $state<'saved' | 'dirty' | 'saving' | 'error'>('saved');
	let job = $state<JobState>('idle');
	let exportState = $state<JobState>('idle');
	let errorMessage = $state('');
	let projectId = $state('');
	let jobId = $state('');
	let canonical = $state<ContractDeck | null>(null);
	const api = new PresentatorApi();
	let zoom = $state(68);
	let note = $state('');
	let notes = $state(['Keep the opening line confident and direct.']);
	let timer: ReturnType<typeof setTimeout>;
	let slide = $derived(deck.slides[slideIndex]);
	let selected = $derived(slide?.elements.find((e) => e.id === selectedId));
	let previewHtml = $derived(
		canonical
			? renderDeckHtml(canonical, {
					title: deck.title,
					slideId: slide?.id,
					assetUrl: (assetId) => projectAssetUrl(projectId, assetId)
				})
			: ''
	);
	onMount(async () => {
		try {
			projectId = localStorage.getItem('presentator-project') ?? '';
			if (!projectId) {
				projectId = (await api.createProject(deck.title)).id;
				localStorage.setItem('presentator-project', projectId);
			}
			const snapshot = await api.getDeck(projectId);
			canonical = snapshot.deck;
			deck = fromContract(snapshot.deck, snapshot.revision);
			selectedId = deck.slides[0]?.elements[0]?.id ?? '';
		} catch (error) {
			errorMessage = messageFor(error, 'API unavailable. Check that the local server is running.');
		}
	});
	function messageFor(error: unknown, fallback: string) {
		return error instanceof ApiError ? error.message : fallback;
	}
	function dirty() {
		save = 'dirty';
		clearTimeout(timer);
		timer = setTimeout(async () => {
			if (!canonical || !projectId) return;
			save = 'saving';
			try {
				const snapshot = await api.saveDeck(projectId, toContract(deck, canonical), deck.revision);
				canonical = snapshot.deck;
				deck.revision = snapshot.revision;
				save = 'saved';
				errorMessage = '';
			} catch (error) {
				save = 'error';
				errorMessage = messageFor(error, 'Save failed. Retry after checking the API.');
			}
		}, 750);
	}
	function patch(value: Partial<DeckElement>) {
		if (selected) {
			Object.assign(selected, value);
			dirty();
		}
	}
	function geometry(key: 'x' | 'y' | 'w' | 'h', value: number) {
		if (selected) {
			Object.assign(selected, updateGeometry(selected, key, value));
			dirty();
		}
	}
	function addSlide() {
		deck.slides.push({
			id: `slide-${Date.now()}`,
			title: 'Untitled slide',
			background: '#fcfbf8',
			elements: []
		});
		slideIndex = deck.slides.length - 1;
		selectedId = '';
		dirty();
	}
	function duplicate() {
		deck = duplicateSlide(deck, slideIndex);
		slideIndex++;
		dirty();
	}
	async function generate() {
		if (job === 'running' || job === 'queued') return;
		try {
			const created = await api.createGeneration(
				projectId,
				'Improve this deck while preserving its theme.'
			);
			jobId = created.id;
			job = created.status;
			const completed = await waitForJob(api, jobId);
			job = completed.status;
			if (completed.status === 'failed')
				errorMessage = completed.error?.message ?? 'Generation failed.';
		} catch (error) {
			job = 'failed';
			errorMessage = messageFor(error, 'Generation failed.');
		}
	}
	async function reviewCandidate() {
		try {
			const candidate = await api.getCandidate(jobId);
			canonical = candidate;
			deck = fromContract(candidate, deck.revision);
			mode = 'preview';
		} catch (error) {
			job = 'failed';
			errorMessage = messageFor(error, 'Candidate unavailable.');
		}
	}
	async function applyCandidate() {
		try {
			const snapshot = await api.applyCandidate(jobId, deck.revision);
			canonical = snapshot.deck;
			deck = fromContract(snapshot.deck, snapshot.revision);
			job = 'idle';
			mode = 'edit';
		} catch (error) {
			job = 'failed';
			errorMessage = messageFor(error, 'Could not apply candidate.');
		}
	}
	async function exportPdf() {
		if (save !== 'saved') {
			errorMessage = 'Wait for autosave before exporting.';
			return;
		}
		try {
			exportState = 'queued';
			const created = await api.createExport(projectId, deck.revision);
			const completed = await waitForJob(api, created.id, undefined, 'export');
			exportState = completed.status;
			if (completed.status !== 'succeeded') {
				errorMessage = completed.error?.message ?? 'Export failed.';
				return;
			}
			const blob = await api.downloadExport(completed.id);
			const href = URL.createObjectURL(blob);
			const anchor = document.createElement('a');
			anchor.href = href;
			anchor.download = `${deck.title}.pdf`;
			anchor.click();
			URL.revokeObjectURL(href);
		} catch (error) {
			exportState = 'failed';
			errorMessage = messageFor(error, 'Export failed.');
		}
	}
	function addNote() {
		if (note.trim()) {
			notes = [...notes, note.trim()];
			note = '';
			dirty();
		}
	}
	const style = (e: DeckElement) =>
		`left:${e.x}%;top:${e.y}%;width:${e.w}%;height:${e.h}%;color:${e.fill};${e.kind === 'shape' ? `background:${e.fill}` : ''}`;
</script>

<svelte:head><title>{deck.title} · Presentator</title></svelte:head>
<div class="app" class:preview={mode === 'preview'}>
	<header>
		<div class="brand">
			<b>P</b><strong>Presentator</strong><i>/</i><button>{deck.title}</button>
		</div>
		<div class="history">
			<button aria-label="Undo">↶</button><button aria-label="Redo" disabled>↷</button>
		</div>
		<div class="modes">
			<button class:active={mode === 'edit'} onclick={() => (mode = 'edit')}>Edit</button><button
				class:active={mode === 'preview'}
				onclick={() => (mode = 'preview')}>Preview</button
			>
		</div>
		<div class="actions">
			<span class:pending={save !== 'saved'}
				>● {save === 'saved'
					? `Saved · r${deck.revision}`
					: save === 'dirty'
						? 'Unsaved changes'
						: save === 'saving'
							? 'Saving…'
							: 'Save failed'}</span
			><button class="plain">Share</button><button
				class="dark"
				onclick={exportPdf}
				disabled={exportState === 'queued' || exportState === 'running'}
				>{exportState === 'queued' || exportState === 'running'
					? 'Exporting…'
					: 'Export PDF⌄'}</button
			>
		</div>
	</header>
	<nav class="tools">
		<div>
			<button class="plain" onclick={addSlide}>＋ New slide</button><button><b>T</b> Text</button
			><button>▧ Image</button><button>○ Shape</button><button>▥ Chart</button>
		</div>
		<div>
			<button>▦ Templates</button><button>◈ Brand style</button><button
				class="ai"
				onclick={generate}
				disabled={job === 'queued' || job === 'running'}
				>✦ {job === 'idle'
					? 'Generate with AI'
					: job === 'queued'
						? 'AI queued'
						: job === 'running'
							? 'Building story…'
							: job === 'succeeded'
								? 'Candidate ready'
								: 'Retry AI'}</button
			>
		</div>
	</nav>
	{#if errorMessage}<div class="error-banner" role="alert">
			<span>{errorMessage}</span><button
				onclick={() => (errorMessage = '')}
				aria-label="Dismiss error">×</button
			>
		</div>{/if}
	<main>
		<aside class="slides">
			<div class="aside-head">Slides <small>{deck.slides.length}</small><button>•••</button></div>
			<div class="slide-list">
				{#each deck.slides as item, index (item.id)}<button
						class:selected={index === slideIndex}
						class="slide-row"
						onclick={() => {
							slideIndex = index;
							selectedId = item.elements[0]?.id ?? '';
						}}
						><span>{String(index + 1).padStart(2, '0')}</span><i
							style={`background:${item.background};color:${item.elements[0]?.fill ?? '#20242b'}`}
							>{item.elements.find((e) => e.kind === 'text')?.content ?? 'Untitled slide'}</i
						></button
					>{/each}
			</div>
			<div class="slide-actions">
				<button onclick={addSlide}>＋ Add slide</button><button
					onclick={duplicate}
					aria-label="Duplicate slide">⧉</button
				>
			</div>
		</aside>
		<section class="stage" aria-label="Canvas workspace">
			{#if job === 'succeeded'}<div class="candidate">
					✦ AI candidate is ready <button onclick={reviewCandidate}>Review</button><button
						class="apply"
						onclick={applyCandidate}>Apply</button
					>
				</div>{/if}
			<div class="canvas-wrap" style={`--zoom:${zoom / 100}`}>
				<div
					class="canvas"
					style={`background:${slide.background}`}
					role="application"
					aria-label={`Slide ${slideIndex + 1}: ${slide.title}`}
				>
					{#if mode === 'preview' && previewHtml}
						<iframe class="scene-preview" title={`Preview of ${deck.title}`} srcdoc={previewHtml}
						></iframe>
					{:else}
						{#each slide.elements as element (element.id)}{#if !element.hidden}<button
									class="element"
									class:headline={element.name === 'Headline'}
									class:body={element.id === 'body'}
									class:eyebrow={element.id === 'eyebrow'}
									class:shape={element.kind === 'shape'}
									class:selected-element={element.id === selectedId && mode === 'edit'}
									style={style(element)}
									disabled={mode === 'preview'}
									onclick={() => {
										selectedId = element.id;
										tab = 'design';
									}}
									>{#if element.kind === 'shape'}<b>{element.content}</b><small>teams aligned</small
										>
										<hr
										/>{:else}{element.content}{/if}{#if element.id === selectedId && mode === 'edit'}<i
											class="tl"
										></i><i class="tr"></i><i class="bl"></i><i class="br"></i>{/if}</button
								>{/if}{/each}
						<div class="meta">
							<span>{String(slideIndex + 1).padStart(2, '0')}</span><span>PRESENTATOR</span>
						</div>
					{/if}
				</div>
			</div>
			<div class="zoom">
				<button onclick={() => (zoom = Math.max(40, zoom - 10))}>−</button><span>{zoom}%</span
				><button onclick={() => (zoom = Math.min(100, zoom + 10))}>＋</button><button
					onclick={() => (zoom = 68)}>Fit</button
				>
			</div>
		</section>
		<aside class="inspector">
			<div class="tabs">
				{#each ['design', 'layers', 'comments'] as item (item)}<button
						class:active={tab === item}
						onclick={() => (tab = item as typeof tab)}
						>{item}{item === 'comments' ? ` ${notes.length}` : ''}</button
					>{/each}
			</div>
			{#if tab === 'design'}{#if selected}<div class="selection">
						<b>{selected.kind === 'text' ? 'T' : '◇'}</b><span
							><small>{selected.kind}</small><strong>{selected.name}</strong></span
						><button>•••</button>
					</div>
					<section>
						<h2>Position & size</h2>
						<div class="grid">
							{#each ['x', 'y', 'w', 'h'] as key (key)}<label
									><span>{key.toUpperCase()}</span><input
										aria-label={key.toUpperCase()}
										type="number"
										value={selected[key as 'x']}
										onchange={(e) => geometry(key as 'x', e.currentTarget.valueAsNumber)}
									/></label
								>{/each}
						</div>
						<div class="quick">
							<button>↻ 0°</button><button
								class:active={selected.locked}
								onclick={() => patch({ locked: !selected?.locked })}>⌑ Lock</button
							><button onclick={() => patch({ hidden: !selected?.hidden })}>◉ Visible</button>
						</div>
					</section>
					{#if selected.kind === 'text'}<section>
							<h2>Text</h2>
							<textarea
								aria-label="Text content"
								value={selected.content}
								oninput={(e) => patch({ content: e.currentTarget.value })}></textarea>
							<div class="font">
								<select><option>Inter</option><option>Georgia</option></select><input
									value="72"
									aria-label="Font size"
								/>
							</div>
							<div class="format">
								<button class="active">B</button><button><i>I</i></button><button>≡</button><button
									>☷</button
								>
							</div>
						</section>{/if}
					<section>
						<h2>Appearance</h2>
						<label class="color"
							><input
								type="color"
								value={selected.fill}
								onchange={(e) => patch({ fill: e.currentTarget.value })}
							/><span>{selected.fill.toUpperCase()}</span><b>100%</b></label
						>
					</section>{:else}<p class="empty">
						◇<br />Select an object to edit its properties.
					</p>{/if}{:else if tab === 'layers'}<div class="layers">
					<small>SLIDE {slideIndex + 1}</small
					>{#each [...slide.elements].reverse() as element (element.id)}<button
							class:selected={element.id === selectedId}
							onclick={() => (selectedId = element.id)}
							><span>{element.kind === 'text' ? 'T' : '◇'}</span><b>{element.name}</b><span
								>{element.locked ? '⌑' : ''} {element.hidden ? '○' : '◉'}</span
							></button
						>{/each}
				</div>{:else}<div class="comments">
					<p>Notes stay with this slide and never appear in exports.</p>
					{#each notes as item, index (`${index}-${item}`)}<article>
							<b>YO</b>
							<div>
								<strong>You <small>now</small></strong>
								<p>{item}</p>
								<button>Resolve</button>
							</div>
						</article>{/each}<label
						>Add a comment<textarea bind:value={note} placeholder="Leave a clear note…"
						></textarea></label
					><button class="dark" onclick={addNote}>Post comment</button>
				</div>{/if}
		</aside>
	</main>
	{#if mode === 'preview'}<div class="preview-note">
			Preview · editing locked <button onclick={() => (mode = 'edit')}>Esc to Edit</button>
		</div>{/if}
</div>
