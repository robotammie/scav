import { h } from 'preact';

import Header from './header';
import Home from '../routes/home';

const App = () => (
	<div id="app">
		<Header />
		<Home />
	</div>
)

export default App;
