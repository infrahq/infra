export default function InputDropdown({ 
  label,
  type,
  value,
  placeholder,
  optionType,
  options,
  handleInputChange,
  handleSelectOption
}) {
  return (
    <div>
      {label && <label htmlFor="price" className="block text-sm font-medium text-white">
        {label}
      </label>}
      <div className="relative rounded-md shadow-sm">
        <input
            type={type}
            value={value}
            className="block w-full px-4 py-2 sm:text-sm border border-gray-500 rounded bg-transparent required:border-red-500 focus:outline-none"
            placeholder={placeholder}
            onChange={handleInputChange}
          />
          <div className="absolute inset-y-0 right-2 flex items-center">
          <label htmlFor={optionType} className="sr-only">
            {optionType}
          </label>
          <select
            id={optionType}
            name={optionType}
            onChange={handleSelectOption}
            className="focus:outline-none h-full py-0 pl-2 pr-1 border-transparent bg-transparent text-gray-500 sm:text-sm rounded-md"
          >
            {options.map((option) => (
              <option key={option} value={option}>{option}</option>
            ))}
          </select>
        </div>
      </div>
    </div>
  )
}