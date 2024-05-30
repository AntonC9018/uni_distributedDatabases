package config

func readConfig() (*viper.Viper, error) {
	config := viper.New()
	config.SetConfigName("configuration")
	config.SetConfigType("json")
	config.AddConfigPath(".")
	{
		err := config.ReadInConfig()
		if err != nil {
            return config, err
		}
	}
    return config, nil
}
