import { mount } from 'svelte';
// D-001: tipografia e iconografia do protótipo Nocturne — Inter (única família)
// e Phosphor (classes `ph`/`ph-bold`), ambos self-hosted para o PWA abrir
// offline já com a cara certa.
import '@fontsource-variable/inter';
import '@phosphor-icons/web/regular';
import '@phosphor-icons/web/bold';
import './app.css';
import App from './App.svelte';

const app = mount(App, { target: document.getElementById('app')! });

export default app;
