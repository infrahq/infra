import moment from 'moment'
import React, { useRef, useEffect, useState } from 'react'
import { ChevronLeftIcon, ChevronRightIcon } from '@heroicons/react/outline'

const monthName = [
  'January',
  'February',
  'March',
  'April',
  'May',
  'June',
  'July',
  'August',
  'September',
  'October',
  'November',
  'December',
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
}) {
  let content = []
  //first row with empty spaces
  if (!row) {
    for (let i = 0; i < firstDay; i++) {
      content.push(<td key={`norow-${i}`}></td>)
    }
    content.push(
      <td
        className='relative py-3 px-2 text-center text-gray-800 hover:cursor-pointer hover:text-gray-400 md:px-3'
        onClick={() => onChange(1)}
      >
        {activeDay === 1 &&
        selectedDate.month() === currentMonth &&
        selectedDate.year() === currentYear ? (
          <div className='mx-auto flex h-6 w-6 items-center justify-center rounded-full bg-blue-500 text-white'>
            <span>1</span>
          </div>
        ) : (
          1
        )}
      </td>
    )
    let len = 7 - content.length
    for (let i = 1; i <= len; i++) {
      content.push(
        <React.Fragment key={i}>
          <td
            className='relative py-3 px-2 text-center text-gray-800 hover:cursor-pointer hover:text-gray-400 md:px-3'
            onClick={() => onChange(i + 1)}
          >
            {activeDay === i + 1 &&
            selectedDate.month() === currentMonth &&
            selectedDate.year() === currentYear ? (
              <div className='mx-auto flex h-6 w-6 items-center justify-center rounded-full bg-blue-500 text-white'>
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
      content.push(
        <React.Fragment key={`${row}-${i}`}>
          <td
            className='relative py-3 px-2 text-center text-gray-800 hover:cursor-pointer hover:text-gray-400 md:px-3'
            onClick={() => onChange(i + (7 * row - firstDay))}
          >
            {activeDay === i + (7 * row - firstDay) &&
            selectedDate.month() === currentMonth &&
            selectedDate.year() === currentYear ? (
              <div className='mx-auto flex h-6 w-6 items-center justify-center rounded-full bg-blue-500 text-white'>
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

export default function Calendar({ selectedDate, onChange }) {
  const [activeMonth, setActiveMonth] = useState(
    moment(selectedDate, 'DD/MM/YYYY').month()
  )
  const [activeMonthString, setActiveMonthString] = useState('')
  const [activeYear, setActiveYear] = useState(
    moment(selectedDate, 'DD/MM/YYYY').year()
  )
  const [firstDayInMonth, setFirstDayInMonth] = useState([])

  const previousMonth = useRef(null)

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

  return (
    <div className='border border-gray-200 bg-white p-4 md:w-96 md:rounded md:shadow-lg'>
      <div className='w-full rounded'>
        <div className='mb-4 flex items-center justify-between'>
          <div className='text-left text-xl font-bold text-black'>
            {`${activeMonthString} ${activeYear}`}
          </div>
          <div className='flex space-x-4'>
            <button
              type='button'
              onClick={() => {
                if (previousMonth.current === 0) {
                  setActiveYear(activeYear - 1)
                  setActiveMonth(11)
                } else {
                  setActiveMonth(activeMonth - 1)
                }
              }}
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
          <table className='w-full text-gray-800'>
            <thead>
              <tr>
                <th className='py-3 px-2 md:px-3 '>S</th>
                <th className='py-3 px-2 md:px-3 '>M</th>
                <th className='py-3 px-2 md:px-3 '>T</th>
                <th className='py-3 px-2 md:px-3 '>W</th>
                <th className='py-3 px-2 md:px-3 '>T</th>
                <th className='py-3 px-2 md:px-3 '>F</th>
                <th className='py-3 px-2 md:px-3 '>S</th>
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
                    activeDay={moment(selectedDate, 'DD/MM/YYYY').date()}
                    onChange={e => {
                      const selectedDate = moment(
                        `${activeYear}-${activeMonth + 1}-${e}`,
                        'YYYY/MM/DD'
                      ).format('DD/MM/YYYY')
                      onChange(selectedDate)
                    }}
                    selectedDate={moment(selectedDate, 'DD/MM/YYYY')}
                  />
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}
