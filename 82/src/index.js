var wantKeyCodes = {
	112: {id: '2TAOizOnNPo', title: 'The Fast and the Furious'},
	113: {id: 'F_VIM03DXWI', title: '2 Fast 2 Furious'},
	114: {id: 'p8HQ2JLlc4E', title: 'The Fast and the Furious: Tokyo Drift'},
	115: {id: '9eBR_u2iRus', title: 'Fast & Furious'},
	116: {id: 'mw2AqdB5EVA', title: 'Fast Five'},
	117: {id: 'dKi5XoeTN0k', title: 'Fast & Furious 6'},
	118: {id: 'Skpu5HaVkOc', title: 'Furious 7'},
	119: {id: 'uisBaTkQAEs', title: 'The Fate of the Furious'},
	120: {id: 'FUK2kdPsBws', title: 'F9'},
	121: {id: 'Qau7AQIogVs', title: 'Fast X'},
	// TODO: 11
	122: {id: '9SA7FaKxZVI', title: 'Fast & Furious Presents: Hobbs & Shaw'},
};

var showing = false;

function onKeyDown(e) {
	var keyCode = e.keyCode || e.which;
	if (!(keyCode in wantKeyCodes) && keyCode != 32) {
		return
	}

	e.preventDefault();

	var titleEl = document.getElementById('movie-title');
	var trailerEl = document.getElementById('movie-trailer');
	if (keyCode == 32) {
		trailerEl.classList.toggle('animate-spin-slow');
		return;
	}

	var movie = wantKeyCodes[keyCode];
	var url = `https://www.youtube.com/embed/${movie.id}?&autoplay=1`;
	titleEl.innerText = movie.title;
	trailerEl.innerHTML =
		`<iframe width="800" height="600" src="${url}" `
		+ `allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"`
		+ `allowFullScreen frameBorder="0" />`;

	if (!showing) {
		showing = true;
		var msgEl = document.getElementById('party-message');
		msgEl.classList.remove('hidden');
	}
}

window.onload = function main() {
	document.addEventListener('keydown', onKeyDown);
};

