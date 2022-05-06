import { h } from 'preact';
import { Router } from 'preact-router';

import Header from './header';
import Home from '../routes/home';

const App = () => (
	<div id="app">
		<Header />
		<Router>
			<Home path="/" />
		</Router>
	</div>
)

export default App;
