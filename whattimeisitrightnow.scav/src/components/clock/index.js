import { h,
	Component } from 'preact';
import style from './style.css';

const EPOCH = new Date(2022, 4, 5);
const FIRST_DAY = new Date(2012, 0, 1);

function convert(d) {
	return new Date(FIRST_DAY.getTime() + ((d.getTime() - EPOCH.getTime()) * 96))
}

const MONTHS = [
	"January",
	"February",
	"March",
	"April",
	"May",
	"June",
	"July",
	"August",
	"September",
	"October",
	"November",
	"December"
];

class Clock extends Component {
	state = {
		time: new Date()
	};

	componentDidMount() {
		this.timerID = setInterval(() => this.tick(), 100);
	}

	componentWillUnmount() {
		clearInterval(this.timerID);
	}

	tick() {
		this.setState({
			time: new Date()
		});
	}

	render() {
		let time = convert(this.state.time);
		let year = time.getFullYear();
		let month = MONTHS[time.getMonth()];
		let date = time.getDate();

		let hour = time.getHours();
		if (hour < 10) { hour = "0" + hour; }

		let minutes = time.getMinutes();
		if (minutes < 10) { minutes = "0" + minutes; }

		let seconds = time.getSeconds();
		if (seconds < 10) { seconds = "0" + seconds; }

		return (
			<div class={style.clock}>
				<div class={style.date_row}>
					<span class="date-tile date">{ date }</span>
					<span class="date-tile month">{ month }</span>
					<span class="date-tile year">{ year }</span>
				</div>
				<div class={style.time_row}>
					<span class={style.hour}>{ hour }</span>
					<span class="minutes">{ minutes }</span>
					<span class="seconds">{ seconds }</span>
				</div>
			</div>
		);
	}
}

export default Clock;
