import moment from 'moment'
import React, { useRef, useEffect, useState } from 'react'
import { ChevronLeftIcon, ChevronRightIcon } from '@heroicons/react/outline'

const monthName = [
  { id: 0, long: 'January', short: 'Jan' },
  { id: 1, long: 'February', short: 'Feb' },
  { id: 2, long: 'March', short: 'Mar' },
  { id: 3, long: 'April', short: 'Apr' },
  { id: 4, long: 'May', short: 'May' },
  { id: 5, long: 'June', short: 'Jun' },
  { id: 6, long: 'July', short: 'Jul' },
  { id: 7, long: 'August', short: 'Aug' },
  { id: 8, long: 'September', short: 'Sept' },
  { id: 9, long: 'October', short: 'Oct' },
  { id: 10, long: 'November', short: 'Nov' },
  { id: 11, long: 'December', short: 'Dec' },
]

const daysOfWeek = [
  { id: 0, title: 'S', value: 'Sunday' },
  { id: 1, title: 'M', value: 'Monday' },
  { id: 2, title: 'T', value: 'Tuesday' },
  { id: 3, title: 'W', value: 'Wednesday' },
  { id: 4, title: 'T', value: 'Thursday' },
  { id: 5, title: 'F', value: 'Friday' },
  { id: 6, title: 'S', value: 'Sunday' },
]

const timeRange = [
  { name: 'in 30 days', value: '720h' },
  { name: 'in 60 days', value: '1440h' },
  { name: 'in 90 days', value: '2160h' },
  { name: 'in 1 year', value: '8766h' },
]

function CalendarRow({
  firstDay,
  lastDayInMonth,
  row,
  currentMonth,
  currentYear,
  onChange,
  activeDay,
  selectedDate,
  extensionHour,
}) {
  const earliestMaxExtensionTime = moment().add(extensionHour, 'h')
  console.log(earliestMaxExtensionTime)

  let content = []
  //first row with empty spaces
  if (!row) {
    for (let i = 0; i < firstDay; i++) {
      content.push(<td key={`norow-${i}`}></td>)
    }

    const isBefore = moment(
      `${currentYear}-${currentMonth + 1}-1`,
      'YYYY-MM-DD'
    ).isBefore(earliestMaxExtensionTime)

    content.push(
      <td
        className={`relative py-2 px-1 text-center hover:cursor-pointer hover:text-gray-400 sm:px-2 ${
          isBefore ? 'text-gray-400 hover:cursor-not-allowed' : ''
        }`}
        onClick={() => {
          if (!isBefore) {
            onChange(1)
          }
        }}
        key='first-day-in-month'
      >
        {activeDay === 1 &&
        selectedDate.month() === currentMonth &&
        selectedDate.year() === currentYear ? (
          <div className='mx-auto flex h-5 w-5 items-center justify-center rounded-full bg-blue-500 text-white'>
            <span>1</span>
          </div>
        ) : (
          <span>1</span>
        )}
      </td>
    )
    let len = 7 - content.length
    for (let i = 1; i <= len; i++) {
      const isBefore = moment(
        `${currentYear}-${currentMonth + 1}-${i + 1}`,
        'YYYY-MM-DD'
      ).isBefore(earliestMaxExtensionTime)

      content.push(
        <React.Fragment key={i + 1}>
          <td
            className={`relative py-2 px-1 text-center hover:cursor-pointer hover:text-gray-400 sm:px-2 ${
              isBefore ? 'text-gray-400 hover:cursor-not-allowed' : ''
            }`}
            onClick={() => {
              if (!isBefore) {
                onChange(i + 1)
              }
            }}
          >
            {activeDay === i + 1 &&
            selectedDate.month() === currentMonth &&
            selectedDate.year() === currentYear ? (
              <div className='mx-auto flex h-5 w-5 items-center justify-center rounded-full bg-blue-500 text-white'>
                <span>{i + 1}</span>
              </div>
            ) : (
              <span>{i + 1}</span>
            )}
          </td>
        </React.Fragment>
      )
    }

    return <>{content}</>
  }
  //other rows
  for (let i = 1; i <= 7; i++) {
    if (i + (7 * row - firstDay) <= lastDayInMonth) {
      const isBefore = moment(
        `${currentYear}-${currentMonth + 1}-${i + (7 * row - firstDay)}`,
        'YYYY-MM-DD'
      ).isBefore(earliestMaxExtensionTime)

      content.push(
        <React.Fragment key={`${row}-${i}`}>
          <td
            className={`relative py-2 px-1 text-center hover:cursor-pointer hover:text-gray-400 sm:px-2 ${
              isBefore ? 'text-gray-400 hover:cursor-not-allowed' : ''
            }`}
            onClick={() => {
              if (!isBefore) {
                onChange(i + (7 * row - firstDay))
              }
            }}
          >
            {activeDay === i + (7 * row - firstDay) &&
            selectedDate.month() === currentMonth &&
            selectedDate.year() === currentYear ? (
              <div className='mx-auto flex h-5 w-5 items-center justify-center rounded-full bg-blue-500 text-white'>
                <span>{i + (7 * row - firstDay)}</span>
              </div>
            ) : (
              <span>{i + (7 * row - firstDay)}</span>
            )}
          </td>
        </React.Fragment>
      )
    }
  }
  return <>{content}</>
}

export default function Calendar({ selectedDate, onChange, extensionHour }) {
  const [activeMonth, setActiveMonth] = useState(null)
  const [activeMonthString, setActiveMonthString] = useState({})
  const [activeYear, setActiveYear] = useState(null)
  const [firstDayInMonth, setFirstDayInMonth] = useState([])

  const previousMonth = useRef(null)

  const today = moment().startOf('day')

  useEffect(() => {
    const newFirstDayInMonth = []

    for (let i = 1; i <= 12; i++) {
      newFirstDayInMonth.push(
        moment(`${activeYear}-${i}-1`, 'YYYY/MM/DD').day()
      )
    }

    setFirstDayInMonth(newFirstDayInMonth)
  }, [activeYear])

  useEffect(() => {
    setActiveMonthString(monthName[activeMonth])
    previousMonth.current = activeMonth
  }, [activeMonth])

  useEffect(() => {
    const earliestMaxExtensionTime = moment()
      .add(extensionHour, 'h')
      .format('YYYY/MM/DD')

    if (selectedDate === 'YYYY/MM/DD') {
      setActiveMonth(moment(earliestMaxExtensionTime, 'YYYY/MM/DD').month())
      setActiveYear(moment(earliestMaxExtensionTime, 'YYYY/MM/DD').year())
    } else {
      setActiveMonth(moment(selectedDate, 'YYYY/MM/DD').month())
      setActiveYear(moment(selectedDate, 'YYYY/MM/DD').year())
    }
  }, [selectedDate])

  return (
    <div className='w-60 border border-gray-200 bg-white p-3 sm:w-96 sm:rounded sm:p-4 sm:shadow-lg'>
      <div className='flex w-full items-center divide-x rounded'>
        {/* date range selection */}
        <div className='w-1/4 space-y-4 px-2 text-xs text-gray-900'>
          {timeRange.map(range => (
            <button
              className='hover:text-gray-300 disabled:text-gray-400 disabled:hover:cursor-not-allowed'
              type='button'
              key={range.value}
              onClick={() => {
                const newSelect = moment()
                  .add(
                    parseInt(range.value),
                    range.value.charAt(range.value.length - 1)
                  )
                  .format('YYYY/MM/DD')

                onChange(newSelect)
              }}
              disabled={parseInt(range.value) < extensionHour}
            >
              {range.name}
            </button>
          ))}
        </div>
        {/* datepicker */}
        <div className='w-3/4 flex-col px-6'>
          <div className='mb-4 flex items-center justify-between'>
            <div className='hidden text-left text-sm font-bold text-gray-700 sm:flex'>
              {`${activeMonthString?.long} ${activeYear}`}
            </div>
            <div className='flex text-left text-sm font-bold text-gray-700 sm:hidden'>
              {`${activeMonthString?.short} ${String(activeYear).slice(-2)}`}
            </div>
            <div className='flex space-x-4'>
              <button
                className='disabled:cursor-not-allowed disabled:opacity-30'
                type='button'
                onClick={() => {
                  if (previousMonth.current === 0) {
                    setActiveYear(activeYear - 1)
                    setActiveMonth(11)
                  } else {
                    setActiveMonth(activeMonth - 1)
                  }
                }}
                disabled={
                  previousMonth.current === today.month() &&
                  activeYear === today.year()
                }
              >
                <ChevronLeftIcon className='h-4 w-4' aria-hidden='true' />
              </button>
              <button
                type='button'
                onClick={() => {
                  if (previousMonth.current === 11) {
                    setActiveYear(activeYear + 1)
                    setActiveMonth(0)
                  } else {
                    setActiveMonth(activeMonth + 1)
                  }
                }}
              >
                <ChevronRightIcon className='h-4 w-4' aria-hidden='true' />
              </button>
            </div>
          </div>
          <div className='-mx-2'>
            <table className='w-full text-2xs font-normal text-gray-800'>
              <thead>
                <tr>
                  {daysOfWeek.map(day => (
                    <th
                      key={`${day.id}-${day.value}`}
                      className='py-2 px-1 text-2xs font-semibold sm:px-2'
                    >
                      {day.title}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {[0, 1, 2, 3, 4, 5].map(row => (
                  <tr key={row}>
                    <CalendarRow
                      firstDay={firstDayInMonth[activeMonth]}
                      lastDayInMonth={new Date(
                        activeYear,
                        activeMonth + 1,
                        0
                      ).getDate()}
                      row={row}
                      currentMonth={activeMonth}
                      currentYear={activeYear}
                      activeDay={moment(selectedDate, 'YYYY/MM/DD').date()}
                      extensionHour={extensionHour}
                      onChange={e => {
                        const newSelectedDate = moment(
                          `${activeYear}-${activeMonth + 1}-${e}`,
                          'YYYY/MM/DD'
                        ).format('YYYY/MM/DD')
                        onChange(newSelectedDate)
                      }}
                      selectedDate={moment(selectedDate, 'YYYY/MM/DD')}
                    />
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  )
}
