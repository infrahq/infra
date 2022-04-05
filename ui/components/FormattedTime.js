import PropTypes from 'prop-types'

const FormattedTime = ({ time }) => {
  const dateTime = new Date(time * 1000)

  const intervals = [
    { label: 'year', seconds: 31536000 },
    { label: 'month', seconds: 2592000 },
    { label: 'day', seconds: 86400 },
    { label: 'hour', seconds: 3600 },
    { label: 'minute', seconds: 60 },
    { label: 'second', seconds: 1 }
  ]

  function timeSince (date) {
    console.log(date)
    const seconds = Math.floor((Date.now() - date) / 1000)
    console.log(seconds)
    console.log(Date.now())
    console.log(Date.parse(date))
    const interval = intervals.find(i => i.seconds < seconds)
    const count = Math.floor(seconds / interval.seconds)
    return `${count} ${interval.label}${count !== 1 ? 's' : ''} ago`
  }

  return (
    <>{timeSince(dateTime)}</>
  )
}

FormattedTime.prototype = {
  time: PropTypes.number.isRequired
}

export default FormattedTime
