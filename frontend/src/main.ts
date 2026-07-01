import { mount } from 'svelte'
import App from './App.svelte'
import 'svelte-material-ui/bare.css'
import './app.css'

mount(App, { target: document.getElementById('app')! })
