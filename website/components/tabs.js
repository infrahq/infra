import { useState, createContext, useEffect, useRef } from 'react'

export const TabContext = createContext()

export default function Tabs({ labels, children }) {
  const [currentTab, setCurrentTab] = useState(labels[0])
  const tabsRef = useRef({})
  const activeRef = useRef(null)

  useEffect(() => {
    function resize() {
      const tabElement = tabsRef.current[currentTab]
      activeRef.current.style.transform = `translateX(${tabElement.offsetLeft}px) translateY(${tabElement.offsetTop}px)`
      activeRef.current.style.width = `${tabElement.offsetWidth}px`
      activeRef.current.style.height = `${tabElement.offsetHeight}px`
    }

    resize()

    window.addEventListener('resize', resize)
    return () => {
      window.removeEventListener('resize', resize)
    }
  }, [currentTab])

  return (
    <TabContext.Provider value={currentTab}>
      <div className='relative flex flex-wrap'>
        <div
          ref={activeRef}
          className={`absolute h-4 w-4 rounded-lg bg-blue-600/5 ${
            activeRef.current ? 'transition-all' : ''
          }`}
        ></div>
        {labels.map(label => (
          <button
            key={label}
            ref={e => (tabsRef.current[label] = e)}
            className={`${
              label === currentTab
                ? ' text-blue-700 hover:text-blue-700'
                : 'text-gray-500 hover:text-gray-800'
            } rounded-lg px-7 py-1.5 text-sm font-medium leading-loose transition-colors`}
            onClick={() => setCurrentTab(label)}
          >
            {label}
          </button>
        ))}
      </div>
      {children}
    </TabContext.Provider>
  )
}
