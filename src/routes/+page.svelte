<script lang="ts">
	import { onMount } from 'svelte';
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
	let save = $state<'saved' | 'dirty' | 'saving'>('saved');
	let job = $state<JobState>('idle');
	let zoom = $state(68);
	let note = $state('');
	let notes = $state(['Keep the opening line confident and direct.']);
	let timer: ReturnType<typeof setTimeout>;
	let slide = $derived(deck.slides[slideIndex]);
	let selected = $derived(slide?.elements.find((e) => e.id === selectedId));
	onMount(() => {
		const raw = localStorage.getItem('presentator-deck');
		if (raw)
			try {
				deck = JSON.parse(raw);
			} catch {
				localStorage.removeItem('presentator-deck');
			}
	});
	function dirty() {
		save = 'dirty';
		clearTimeout(timer);
		timer = setTimeout(() => {
			save = 'saving';
			setTimeout(() => {
				localStorage.setItem('presentator-deck', JSON.stringify(deck));
				save = 'saved';
			}, 450);
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
	function generate() {
		if (job === 'running' || job === 'queued') return;
		job = 'queued';
		setTimeout(() => (job = 'running'), 600);
		setTimeout(() => (job = 'succeeded'), 2400);
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
				>● {save === 'saved' ? 'Saved' : save === 'dirty' ? 'Unsaved changes' : 'Saving…'}</span
			><button class="plain">Share</button><button class="dark">Export PDF⌄</button>
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
					✦ AI candidate is ready <button onclick={() => (job = 'idle')}>Review</button><button
						class="apply"
						onclick={() => (job = 'idle')}>Apply</button
					>
				</div>{/if}
			<div class="canvas-wrap" style={`--zoom:${zoom / 100}`}>
				<div
					class="canvas"
					style={`background:${slide.background}`}
					role="application"
					aria-label={`Slide ${slideIndex + 1}: ${slide.title}`}
				>
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
								>{#if element.kind === 'shape'}<b>{element.content}</b><small>teams aligned</small>
									<hr
									/>{:else}{element.content}{/if}{#if element.id === selectedId && mode === 'edit'}<i
										class="tl"
									></i><i class="tr"></i><i class="bl"></i><i class="br"></i>{/if}</button
							>{/if}{/each}
					<div class="meta">
						<span>{String(slideIndex + 1).padStart(2, '0')}</span><span>PRESENTATOR</span>
					</div>
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
